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
	desiredInstances, desiredIngresses, images, err := r.desiredResources()
	if err != nil {
		return Plan{}, err
	}

	var actions []Action
	for _, image := range images {
		action, err := r.planImage(ctx, image)
		if err != nil {
			return Plan{}, err
		}
		actions = append(actions, action)
	}

	existingInstances, err := r.listComposeInstances(ctx)
	if err != nil {
		return Plan{}, err
	}
	allInstances, err := r.client.Instances.List(ctx, hypeman.InstanceListParams{}, r.opts...)
	if err != nil {
		return Plan{}, err
	}
	for _, inst := range desiredInstances {
		actions = append(actions, planInstanceAction(inst, existingInstances, *allInstances))
	}

	existingIngresses, err := r.listComposeIngresses(ctx)
	if err != nil {
		return Plan{}, err
	}
	allIngresses, err := r.client.Ingresses.List(ctx, hypeman.IngressListParams{}, r.opts...)
	if err != nil {
		return Plan{}, err
	}
	for _, ingress := range desiredIngresses {
		actions = append(actions, planIngressAction(ingress, existingIngresses, *allIngresses))
	}

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
		case "replace":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[replace] %s %s\n", action.Type, action.Name)
			}
			if err := r.applyReplace(ctx, action, opts); err != nil {
				return result, err
			}
		case "unchanged":
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[skip] %s %s unchanged\n", action.Type, action.Name)
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

func (r *Runner) Down(ctx context.Context, verbose bool) (Plan, error) {
	instances, err := r.listComposeInstances(ctx)
	if err != nil {
		return Plan{}, err
	}
	ingresses, err := r.listComposeIngresses(ctx)
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
	sortComposeActions(actions)

	result := Plan{
		Name:    r.spec.Name,
		File:    r.file,
		Actions: actions,
		Summary: summarizeComposeActions(actions),
	}
	if len(actions) == 0 {
		_, desiredIngresses, _, err := r.desiredResources()
		if err != nil {
			return Plan{}, err
		}
		for _, ingress := range desiredIngresses {
			result.Actions = append(result.Actions, Action{
				Action:  "skip",
				Type:    "ingress",
				Name:    ingress.Name,
				Service: ingress.Service,
				Reason:  "not found",
			})
		}
		for serviceName := range r.spec.Services {
			result.Actions = append(result.Actions, Action{
				Action:  "skip",
				Type:    "instance",
				Name:    composeInstanceName(r.spec.Name, serviceName),
				Service: serviceName,
				Reason:  "not found",
			})
		}
		sortComposeActions(result.Actions)
		result.Summary = summarizeComposeActions(result.Actions)
		return result, nil
	}

	for i := range actions {
		action := &actions[i]
		if verbose {
			fmt.Fprintf(os.Stderr, "[delete] %s %s\n", action.Type, action.Name)
		}
		switch action.Type {
		case "ingress":
			if err := r.client.Ingresses.Delete(ctx, action.ingressID, r.opts...); err != nil && !isHTTPNotFound(err) {
				return result, err
			}
		case "instance":
			if err := r.client.Instances.Delete(ctx, action.instanceID, r.opts...); err != nil && !isHTTPNotFound(err) {
				return result, err
			}
		}
	}

	return result, nil
}

func (r *Runner) applyCreate(ctx context.Context, action *Action, opts UpOptions) error {
	switch action.Type {
	case "image":
		return r.ensureImageReady(ctx, action.Name, opts.Verbose)
	case "instance":
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
			if err := r.client.Instances.Delete(ctx, action.instanceID, r.opts...); err != nil && !isHTTPNotFound(err) {
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

func planIngressAction(desired desiredIngress, owned []hypeman.Ingress, all []hypeman.Ingress) Action {
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

func isHTTPNotFound(err error) bool {
	apiErr, ok := err.(*hypeman.Error)
	return ok && apiErr.Response != nil && apiErr.Response.StatusCode == 404
}

func sortComposeActions(actions []Action) {
	order := map[string]int{
		"image":    0,
		"ingress":  1,
		"instance": 2,
	}
	sort.SliceStable(actions, func(i, j int) bool {
		if order[actions[i].Type] != order[actions[j].Type] {
			return order[actions[i].Type] < order[actions[j].Type]
		}
		return actions[i].Name < actions[j].Name
	})
}
