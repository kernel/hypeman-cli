package compose

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kernel/hypeman-go"
)

func (r *Runner) Plan(ctx context.Context) (Plan, error) {
	desiredBuilds, desiredVolumes, desiredInstances, desiredIngresses, images, err := r.desiredResources()
	if err != nil {
		return Plan{}, err
	}

	var actions []Action
	for _, build := range desiredBuilds {
		action, err := r.planBuild(ctx, build)
		if err != nil {
			return Plan{}, err
		}
		if action.buildInput != nil && action.buildInput.ImageRef != "" {
			if err := updateDesiredInstanceImage(desiredInstances, r.spec.Name, build.Service, action.buildInput.ImageRef); err != nil {
				return Plan{}, err
			}
		}
		actions = append(actions, action)
	}
	for _, image := range images {
		action, err := r.planImage(ctx, image)
		if err != nil {
			return Plan{}, err
		}
		actions = append(actions, action)
	}

	existingVolumes, err := r.listComposeVolumes(ctx)
	if err != nil {
		return Plan{}, err
	}
	allVolumes, err := r.client.Volumes.List(ctx, hypeman.VolumeListParams{}, r.opts...)
	if err != nil {
		return Plan{}, err
	}
	desiredVolumeNames := map[string]struct{}{}
	for _, volume := range desiredVolumes {
		desiredVolumeNames[volume.Name] = struct{}{}
		actions = append(actions, planVolumeAction(volume, existingVolumes, *allVolumes))
	}
	// Retained volumes are never pruned by up/plan, even when they are no
	// longer declared. Deleting their data requires `compose down --volumes`.
	for _, vol := range existingVolumes {
		if _, desired := desiredVolumeNames[vol.Name]; desired {
			continue
		}
		actions = append(actions, Action{
			Action:   "skip",
			Type:     "volume",
			Name:     vol.Name,
			Reason:   "retained: not declared in compose file (use `compose down --volumes` to delete)",
			volumeID: vol.ID,
		})
	}

	existingInstances, err := r.listComposeInstances(ctx)
	if err != nil {
		return Plan{}, err
	}
	allInstances, err := r.client.Instances.List(ctx, hypeman.InstanceListParams{}, r.opts...)
	if err != nil {
		return Plan{}, err
	}
	var instanceActions []Action
	for _, inst := range desiredInstances {
		instanceActions = append(instanceActions, planInstanceAction(inst, existingInstances, *allInstances))
	}

	existingIngresses, err := r.listComposeIngresses(ctx)
	if err != nil {
		return Plan{}, err
	}
	allIngresses, err := r.client.Ingresses.List(ctx, hypeman.IngressListParams{}, r.opts...)
	if err != nil {
		return Plan{}, err
	}
	desiredIngressNames := desiredIngressNamesByService(desiredIngresses)
	var ingressActions []Action
	for _, ingress := range desiredIngresses {
		ingressActions = append(ingressActions, planIngressAction(ingress, existingIngresses, *allIngresses, desiredIngressNames[ingress.Service]))
	}

	// Prune owned instances and ingresses that no desired resource claims.
	// Unmanaged resources (no compose ownership tags) are never touched.
	// Prune deletes are planned before instance/ingress creates and replaces
	// (and Up applies actions in plan order) so a pruned resource frees its
	// unique keys — names, ingress hostnames — before a new resource reuses
	// them. This mirrors applyReplace's delete-then-create.
	claimed := make([]Action, 0, len(instanceActions)+len(ingressActions))
	claimed = append(claimed, instanceActions...)
	claimed = append(claimed, ingressActions...)
	actions = append(actions, pruneActions(existingInstances, existingIngresses, claimed)...)
	actions = append(actions, instanceActions...)
	actions = append(actions, ingressActions...)

	return Plan{
		Name:    r.spec.Name,
		File:    r.file,
		Actions: actions,
		Summary: summarizeComposeActions(actions),
	}, nil
}

func (r *Runner) Up(ctx context.Context, opts UpOptions) (Plan, error) {
	result, err := r.Plan(ctx)
	if err != nil {
		return Plan{}, err
	}
	if blockers := conflictBlockers(result.Actions); len(blockers) > 0 {
		return result, fmt.Errorf("conflicts found:\n%s", strings.Join(blockers, "\n"))
	}
	if blockers := replacementBlockers(result.Actions, opts.Replace); len(blockers) > 0 {
		return result, fmt.Errorf("replace required:\n%s\n\nRun again with --replace to recreate changed resources.", strings.Join(blockers, "\n"))
	}

	// Fresh volume-name lookup for this apply pass; the first instance create
	// populates it (after volume creates have run) and later creates reuse it.
	r.volumeIDsByName = nil

	for i := range result.Actions {
		action := &result.Actions[i]
		switch action.Action {
		case "create":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[create] %s %s\n", action.Type, action.Name)
			}
			if err := r.applyCreate(ctx, action, opts); err != nil {
				return result, err
			}
			if action.Type == "build" {
				if err := updatePlannedInstanceImage(result.Actions, r.spec.Name, action.Service, action.Name); err != nil {
					return result, err
				}
			}
		case "replace":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[replace] %s %s\n", action.Type, action.Name)
			}
			if err := r.applyReplace(ctx, action, opts); err != nil {
				return result, err
			}
		case "delete":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[delete] %s %s\n", action.Type, action.Name)
			}
			if err := r.applyDelete(ctx, action); err != nil {
				return result, err
			}
		case "skip":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[skip] %s %s: %s\n", action.Type, action.Name, action.Reason)
			}
		case "unchanged":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[skip] %s %s unchanged\n", action.Type, action.Name)
			}
			if action.Type == "build" && action.buildInput != nil && action.buildInput.ImageRef != "" {
				action.Name = action.buildInput.ImageRef
				if err := updatePlannedInstanceImage(result.Actions, r.spec.Name, action.Service, action.Name); err != nil {
					return result, err
				}
				if err := r.ensureImageReady(ctx, action.Name, opts.Verbose); err != nil {
					return result, err
				}
			}
			if action.Type == "image" {
				if err := r.ensureImageReady(ctx, action.Name, opts.Verbose); err != nil {
					return result, err
				}
			}
		case "conflict":
			return result, fmt.Errorf("%s %s already exists without compose ownership tags", action.Type, action.Name)
		}
	}

	return result, nil
}

// Down deletes instances and ingresses owned by the compose file. Retained
// volumes are kept by default and reported as skipped; they are only deleted
// when DownOptions.Volumes is set (`compose down --volumes`), which destroys
// their data.
func (r *Runner) Down(ctx context.Context, opts DownOptions) (Plan, error) {
	instances, err := r.listComposeInstances(ctx)
	if err != nil {
		return Plan{}, err
	}
	ingresses, err := r.listComposeIngresses(ctx)
	if err != nil {
		return Plan{}, err
	}
	volumes, err := r.listComposeVolumes(ctx)
	if err != nil {
		return Plan{}, err
	}

	var actions []Action
	for _, ing := range ingresses {
		actions = append(actions, Action{
			Action:    "delete",
			Type:      "ingress",
			Name:      ing.Name,
			Service:   ing.Tags[composeTagService],
			Reason:    "owned by compose file",
			ingressID: ing.ID,
		})
	}
	for _, inst := range instances {
		actions = append(actions, Action{
			Action:     "delete",
			Type:       "instance",
			Name:       inst.Name,
			Service:    inst.Tags[composeTagService],
			Reason:     "owned by compose file",
			instanceID: inst.ID,
		})
	}
	for _, vol := range volumes {
		if opts.Volumes {
			actions = append(actions, Action{
				Action:   "delete",
				Type:     "volume",
				Name:     vol.Name,
				Reason:   "owned by compose file; --volumes destroys retained data",
				volumeID: vol.ID,
			})
			continue
		}
		actions = append(actions, Action{
			Action:   "skip",
			Type:     "volume",
			Name:     vol.Name,
			Reason:   "retained (use --volumes to delete)",
			volumeID: vol.ID,
		})
	}
	sortComposeActions(actions)

	result := Plan{
		Name:    r.spec.Name,
		File:    r.file,
		Actions: actions,
		Summary: summarizeComposeActions(actions),
	}
	if len(instances) == 0 && len(ingresses) == 0 && len(volumes) == 0 {
		for serviceName := range r.spec.Services {
			service := r.spec.Services[serviceName]
			for i := range service.Ingress {
				result.Actions = append(result.Actions, Action{
					Action:  "skip",
					Type:    "ingress",
					Name:    composeIngressName(r.spec.Name, serviceName, i, service.Ingress[i]),
					Service: serviceName,
					Reason:  "not found",
				})
			}
			result.Actions = append(result.Actions, Action{
				Action:  "skip",
				Type:    "instance",
				Name:    composeInstanceName(r.spec.Name, serviceName, service),
				Service: serviceName,
				Reason:  "not found",
			})
		}
		volumeKeys := make([]string, 0, len(r.spec.Volumes))
		for key := range r.spec.Volumes {
			volumeKeys = append(volumeKeys, key)
		}
		sort.Strings(volumeKeys)
		for _, key := range volumeKeys {
			result.Actions = append(result.Actions, Action{
				Action: "skip",
				Type:   "volume",
				Name:   composeVolumeName(r.spec.Name, key, r.spec.Volumes[key]),
				Reason: "not found",
			})
		}
		sortComposeActions(result.Actions)
		result.Summary = summarizeComposeActions(result.Actions)
		return result, nil
	}

	for i := range actions {
		action := &actions[i]
		if action.Action != "delete" {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[skip] %s %s: %s\n", action.Type, action.Name, action.Reason)
			}
			continue
		}
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[delete] %s %s\n", action.Type, action.Name)
		}
		if err := r.applyDelete(ctx, action); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (r *Runner) applyDelete(ctx context.Context, action *Action) error {
	switch action.Type {
	case "ingress":
		if err := r.client.Ingresses.Delete(ctx, action.ingressID, r.opts...); err != nil && !isHTTPNotFound(err) {
			return err
		}
	case "instance":
		if err := r.client.Instances.Delete(ctx, action.instanceID, hypeman.InstanceDeleteParams{}, r.opts...); err != nil && !isHTTPNotFound(err) {
			return err
		}
	case "volume":
		if err := r.client.Volumes.Delete(ctx, action.volumeID, r.opts...); err != nil && !isHTTPNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *Runner) applyCreate(ctx context.Context, action *Action, opts UpOptions) error {
	switch action.Type {
	case "build":
		if action.buildInput == nil {
			return fmt.Errorf("build action %s missing build input", action.Name)
		}
		imageRef, err := r.runBuild(ctx, *action.buildInput, opts.Verbose)
		if err != nil {
			return err
		}
		if imageRef != "" {
			action.Name = imageRef
		}
		if err := r.ensureImageReady(ctx, action.Name, opts.Verbose); err != nil {
			return err
		}
		return nil
	case "image":
		return r.ensureImageReady(ctx, action.Name, opts.Verbose)
	case "volume":
		vol, err := r.client.Volumes.New(ctx, action.volumeInput, r.opts...)
		if err != nil {
			return err
		}
		action.volumeID = vol.ID
	case "instance":
		if err := r.resolveInstanceVolumeIDs(ctx, &action.instanceInput); err != nil {
			return err
		}
		inst, err := r.client.Instances.New(ctx, action.instanceInput, r.opts...)
		if err != nil {
			return err
		}
		action.instanceID = inst.ID
		if opts.Wait {
			return r.waitForInstanceRunning(ctx, inst.ID, opts.WaitTimeout, opts.Verbose)
		}
	case "ingress":
		ing, err := r.client.Ingresses.New(ctx, action.ingressInput, r.opts...)
		if err != nil {
			return err
		}
		action.ingressID = ing.ID
	}
	return nil
}

func (r *Runner) applyReplace(ctx context.Context, action *Action, opts UpOptions) error {
	switch action.Type {
	case "instance":
		if action.instanceID != "" {
			if err := r.client.Instances.Delete(ctx, action.instanceID, hypeman.InstanceDeleteParams{}, r.opts...); err != nil && !isHTTPNotFound(err) {
				return err
			}
		}
	case "ingress":
		if action.ingressID != "" {
			if err := r.client.Ingresses.Delete(ctx, action.ingressID, r.opts...); err != nil && !isHTTPNotFound(err) {
				return err
			}
		}
	}
	createAction := *action
	createAction.Action = "create"
	if err := r.applyCreate(ctx, &createAction, opts); err != nil {
		return err
	}
	action.instanceID = createAction.instanceID
	action.ingressID = createAction.ingressID
	return nil
}

func (r *Runner) ensureImageReady(ctx context.Context, image string, verbose bool) error {
	img, err := r.client.Images.Get(ctx, url.PathEscape(image), r.opts...)
	if err != nil {
		if !isHTTPNotFound(err) {
			return fmt.Errorf("check image %s: %w", image, err)
		}
		img, err = r.client.Images.New(ctx, hypeman.ImageNewParams{Name: image}, r.opts...)
		if err != nil {
			return fmt.Errorf("create image %s: %w", image, err)
		}
	}
	if verbose && img.Status != hypeman.ImageStatusReady {
		fmt.Fprintf(os.Stderr, "[wait] image %s ready\n", image)
	}
	return waitForImageReady(ctx, &r.client, img)
}

func waitForImageReady(ctx context.Context, client *hypeman.Client, img *hypeman.Image) error {
	if img.Status == hypeman.ImageStatusReady {
		return nil
	}
	if img.Status == hypeman.ImageStatusFailed {
		if img.Error != "" {
			return fmt.Errorf("image build failed: %s", img.Error)
		}
		return fmt.Errorf("image build failed")
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			updated, err := client.Images.Get(ctx, url.PathEscape(img.Name))
			if err != nil {
				return fmt.Errorf("failed to check image status: %w", err)
			}
			switch updated.Status {
			case hypeman.ImageStatusReady:
				return nil
			case hypeman.ImageStatusFailed:
				if updated.Error != "" {
					return fmt.Errorf("image build failed: %s", updated.Error)
				}
				return fmt.Errorf("image build failed")
			}
		}
	}
}

func (r *Runner) waitForInstanceRunning(ctx context.Context, instanceID, timeout string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "[wait] instance %s running\n", instanceID)
	}
	params := hypeman.InstanceWaitParams{
		State: hypeman.InstanceWaitParamsStateRunning,
	}
	if timeout != "" {
		params.Timeout = hypeman.Opt(timeout)
	}
	resp, err := r.client.Instances.Wait(ctx, instanceID, params, r.opts...)
	if err != nil {
		return err
	}
	if resp.TimedOut {
		return fmt.Errorf("timed out waiting for instance %s to reach Running", instanceID)
	}
	return nil
}

func (r *Runner) planImage(ctx context.Context, image string) (Action, error) {
	_, err := r.client.Images.Get(ctx, url.PathEscape(image), r.opts...)
	if err == nil {
		return Action{Action: "unchanged", Type: "image", Name: image, Reason: "already exists"}, nil
	}
	if isHTTPNotFound(err) {
		return Action{Action: "create", Type: "image", Name: image, Reason: "not present"}, nil
	}
	return Action{}, fmt.Errorf("check image %s: %w", image, err)
}

func planInstanceAction(desired desiredInstance, owned []hypeman.Instance, all []hypeman.Instance) Action {
	action := Action{
		Type:          "instance",
		Name:          desired.Name,
		Service:       desired.Service,
		instanceInput: desired.Input,
	}
	for _, inst := range owned {
		if inst.Name != desired.Name {
			continue
		}
		action.instanceID = inst.ID
		if inst.Tags[composeTagHash] == desired.Hash {
			action.Action = "unchanged"
			action.Reason = "hash matches"
			return action
		}
		action.Action = "replace"
		if inst.Tags[composeTagHash] == "" {
			action.Reason = "missing compose hash"
		} else {
			action.Reason = "rendered spec changed"
		}
		return action
	}
	for _, inst := range owned {
		if inst.Tags[composeTagService] != desired.Service || inst.Tags[composeTagResource] != composeResourceInstance {
			continue
		}
		action.Action = "replace"
		action.Reason = fmt.Sprintf("name changed from %s", inst.Name)
		action.instanceID = inst.ID
		return action
	}
	for _, inst := range all {
		if inst.Name == desired.Name {
			action.Action = "conflict"
			action.Reason = "name exists without compose ownership"
			action.instanceID = inst.ID
			return action
		}
	}
	action.Action = "create"
	action.Reason = "missing"
	return action
}

// planVolumeAction reconciles one declared retained volume. Volumes are
// immutable once created: a spec change is a conflict that blocks `up` rather
// than a replacement, because replacing a volume would destroy its data.
func planVolumeAction(desired desiredVolume, owned []hypeman.Volume, all []hypeman.Volume) Action {
	action := Action{
		Type:        "volume",
		Name:        desired.Name,
		volumeInput: desired.Input,
	}
	for _, vol := range owned {
		if vol.Name != desired.Name {
			continue
		}
		action.volumeID = vol.ID
		if vol.Tags[composeTagHash] == desired.Hash {
			action.Action = "unchanged"
			action.Reason = "hash matches"
			return action
		}
		action.Action = "conflict"
		action.Reason = "retained volume spec changed; volumes are immutable (restore the declared spec or delete the volume with `compose down --volumes`)"
		return action
	}
	for _, vol := range all {
		if vol.Name == desired.Name {
			action.Action = "conflict"
			if project := vol.Tags[composeTagName]; project != "" {
				action.Reason = fmt.Sprintf("name is owned by a different compose project %q", project)
			} else {
				action.Reason = "name exists without compose ownership"
			}
			action.volumeID = vol.ID
			return action
		}
	}
	action.Action = "create"
	action.Reason = "missing"
	return action
}

// pruneActions returns delete actions for owned instances and ingresses that
// no desired action claims (removed from the compose file or superseded).
// Resources without compose ownership tags are never included.
func pruneActions(ownedInstances []hypeman.Instance, ownedIngresses []hypeman.Ingress, actions []Action) []Action {
	claimedInstances := map[string]struct{}{}
	claimedIngresses := map[string]struct{}{}
	for _, action := range actions {
		if action.instanceID != "" {
			claimedInstances[action.instanceID] = struct{}{}
		}
		if action.ingressID != "" {
			claimedIngresses[action.ingressID] = struct{}{}
		}
		// Ambiguous-rename conflicts carry no single ingressID but still
		// claim their candidates so plan doesn't also propose deleting them.
		for _, id := range action.claimedIngressIDs {
			claimedIngresses[id] = struct{}{}
		}
	}
	var pruned []Action
	for _, inst := range ownedInstances {
		if _, claimed := claimedInstances[inst.ID]; claimed {
			continue
		}
		pruned = append(pruned, Action{
			Action:     "delete",
			Type:       "instance",
			Name:       inst.Name,
			Service:    inst.Tags[composeTagService],
			Reason:     "no longer declared in compose file",
			instanceID: inst.ID,
		})
	}
	for _, ing := range ownedIngresses {
		if _, claimed := claimedIngresses[ing.ID]; claimed {
			continue
		}
		pruned = append(pruned, Action{
			Action:    "delete",
			Type:      "ingress",
			Name:      ing.Name,
			Service:   ing.Tags[composeTagService],
			Reason:    "no longer declared in compose file",
			ingressID: ing.ID,
		})
	}
	sortComposeActions(pruned)
	return pruned
}

func planIngressAction(desired desiredIngress, owned []hypeman.Ingress, all []hypeman.Ingress, desiredServiceIngressNames map[string]struct{}) Action {
	action := Action{
		Type:         "ingress",
		Name:         desired.Name,
		Service:      desired.Service,
		ingressInput: desired.Input,
	}
	for _, ing := range owned {
		if ing.Name != desired.Name {
			continue
		}
		action.ingressID = ing.ID
		if ing.Tags[composeTagHash] == desired.Hash {
			action.Action = "unchanged"
			action.Reason = "hash matches"
			return action
		}
		action.Action = "replace"
		if ing.Tags[composeTagHash] == "" {
			action.Reason = "missing compose hash"
		} else {
			action.Reason = "rendered spec changed"
		}
		return action
	}
	var renameCandidates []hypeman.Ingress
	for _, ing := range owned {
		if ing.Tags[composeTagService] != desired.Service || ing.Tags[composeTagResource] != composeResourceIngress {
			continue
		}
		if _, stillDesired := desiredServiceIngressNames[ing.Name]; stillDesired {
			continue
		}
		renameCandidates = append(renameCandidates, ing)
	}
	if len(renameCandidates) == 1 {
		ing := renameCandidates[0]
		action.Action = "replace"
		action.Reason = fmt.Sprintf("name changed from %s", ing.Name)
		action.ingressID = ing.ID
		return action
	}
	if len(renameCandidates) > 1 {
		action.Action = "conflict"
		action.Reason = "multiple owned ingresses for service have changed names"
		// The conflict blocks Up, but claim the candidates so pruneActions
		// doesn't also plan deletes for them.
		for _, ing := range renameCandidates {
			action.claimedIngressIDs = append(action.claimedIngressIDs, ing.ID)
		}
		return action
	}
	for _, ing := range all {
		if ing.Name == desired.Name {
			action.Action = "conflict"
			action.Reason = "name exists without compose ownership"
			action.ingressID = ing.ID
			return action
		}
	}
	action.Action = "create"
	action.Reason = "missing"
	return action
}

func desiredIngressNamesByService(ingresses []desiredIngress) map[string]map[string]struct{} {
	names := map[string]map[string]struct{}{}
	for _, ingress := range ingresses {
		if names[ingress.Service] == nil {
			names[ingress.Service] = map[string]struct{}{}
		}
		names[ingress.Service][ingress.Name] = struct{}{}
	}
	return names
}

func (r *Runner) listComposeInstances(ctx context.Context) ([]hypeman.Instance, error) {
	instances, err := r.client.Instances.List(ctx, hypeman.InstanceListParams{
		Tags: map[string]string{composeTagName: r.spec.Name},
	}, r.opts...)
	if err != nil {
		return nil, err
	}
	return *instances, nil
}

func (r *Runner) listComposeIngresses(ctx context.Context) ([]hypeman.Ingress, error) {
	ingresses, err := r.client.Ingresses.List(ctx, hypeman.IngressListParams{
		Tags: map[string]string{composeTagName: r.spec.Name},
	}, r.opts...)
	if err != nil {
		return nil, err
	}
	return *ingresses, nil
}

func (r *Runner) listComposeVolumes(ctx context.Context) ([]hypeman.Volume, error) {
	volumes, err := r.client.Volumes.List(ctx, hypeman.VolumeListParams{
		Tags: map[string]string{composeTagName: r.spec.Name},
	}, r.opts...)
	if err != nil {
		return nil, err
	}
	return *volumes, nil
}

// resolveInstanceVolumeIDs swaps the volume names carried in planned mounts
// for the server-assigned volume IDs required by instance creation. Planned
// mounts deliberately carry names so the rendered hash is stable before the
// volume exists; resolution happens here, immediately before creation.
func (r *Runner) resolveInstanceVolumeIDs(ctx context.Context, input *hypeman.InstanceNewParams) error {
	if len(input.Volumes) == 0 {
		return nil
	}
	// The name→ID lookup is listed once per apply pass and shared across
	// instance creates. Volume actions are planned before instance actions,
	// so the first resolve already sees every volume created this pass.
	if r.volumeIDsByName == nil {
		volumes, err := r.listComposeVolumes(ctx)
		if err != nil {
			return err
		}
		r.volumeIDsByName = make(map[string]string, len(volumes))
		for _, vol := range volumes {
			r.volumeIDsByName[vol.Name] = vol.ID
		}
	}
	for i := range input.Volumes {
		name := input.Volumes[i].VolumeID
		id, ok := r.volumeIDsByName[name]
		if !ok {
			return fmt.Errorf("volume %s for instance %s not found (volume creation may have failed)", name, input.Name)
		}
		input.Volumes[i].VolumeID = id
	}
	return nil
}

func replacementBlockers(actions []Action, replace bool) []string {
	if replace {
		return nil
	}
	var blockers []string
	for _, action := range actions {
		if action.Action == "replace" {
			blockers = append(blockers, fmt.Sprintf("  %s %s changed: %s", action.Type, action.Name, action.Reason))
		}
	}
	return blockers
}

func conflictBlockers(actions []Action) []string {
	var blockers []string
	for _, action := range actions {
		if action.Action == "conflict" {
			blockers = append(blockers, fmt.Sprintf("  %s %s: %s", action.Type, action.Name, action.Reason))
		}
	}
	return blockers
}

func summarizeComposeActions(actions []Action) Summary {
	var summary Summary
	for _, action := range actions {
		switch action.Action {
		case "create":
			summary.Create++
		case "replace":
			summary.Replace++
		case "delete":
			summary.Delete++
		case "unchanged":
			summary.Unchanged++
		case "skip":
			summary.Skip++
		case "conflict":
			summary.Conflict++
		}
	}
	return summary
}

func updatePlannedInstanceImage(actions []Action, composeName, serviceName, image string) error {
	if image == "" {
		return nil
	}
	for i := range actions {
		if actions[i].Type != "instance" || actions[i].Service != serviceName {
			continue
		}
		actions[i].instanceInput.Image = image
		actions[i].instanceInput.Tags = nil
		hash, err := shortHash(actions[i].instanceInput)
		if err != nil {
			return err
		}
		actions[i].instanceInput.Tags = composeTags(composeName, serviceName, composeResourceInstance, hash)
	}
	return nil
}

func isHTTPNotFound(err error) bool {
	apiErr, ok := err.(*hypeman.Error)
	return ok && apiErr.Response != nil && apiErr.Response.StatusCode == 404
}

func sortComposeActions(actions []Action) {
	order := map[string]int{
		"image":    0,
		"ingress":  1,
		"instance": 2,
		"volume":   3,
	}
	sort.SliceStable(actions, func(i, j int) bool {
		if order[actions[i].Type] != order[actions[j].Type] {
			return order[actions[i].Type] < order[actions[j].Type]
		}
		return actions[i].Name < actions[j].Name
	})
}
