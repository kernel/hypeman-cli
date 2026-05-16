package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"gopkg.in/yaml.v3"
)

const (
	composeTagName     = "hypeman.compose.name"
	composeTagService  = "hypeman.compose.service"
	composeTagResource = "hypeman.compose.resource"
	composeTagHash     = "hypeman.compose.hash"

	composeResourceInstance = "instance"
	composeResourceIngress  = "ingress"
)

type Runner struct {
	file   string
	spec   composeSpec
	client hypeman.Client
	opts   []option.RequestOption
}

type UpOptions struct {
	Replace     bool
	Wait        bool
	WaitTimeout string
	Verbose     bool
}

type composeSpec struct {
	Version  int                           `json:"version" yaml:"version"`
	Name     string                        `json:"name" yaml:"name"`
	Services map[string]composeServiceSpec `json:"services" yaml:"services"`
}

type composeServiceSpec struct {
	Image      string                   `json:"image" yaml:"image"`
	Entrypoint []string                 `json:"entrypoint,omitempty" yaml:"entrypoint"`
	Cmd        []string                 `json:"cmd,omitempty" yaml:"cmd"`
	Env        map[string]string        `json:"env,omitempty" yaml:"env"`
	Resources  composeResourcesSpec     `json:"resources,omitempty" yaml:"resources"`
	Restart    *composeRestartSpec      `json:"restart,omitempty" yaml:"restart"`
	Health     *composeCheckSpec        `json:"healthcheck,omitempty" yaml:"healthcheck"`
	Ingress    []composeIngressRuleSpec `json:"ingress,omitempty" yaml:"ingress"`
}

type composeResourcesSpec struct {
	Vcpus             int    `json:"vcpus,omitempty" yaml:"vcpus"`
	Memory            string `json:"memory,omitempty" yaml:"memory"`
	OverlaySize       string `json:"overlay_size,omitempty" yaml:"overlay_size"`
	HotplugSize       string `json:"hotplug_size,omitempty" yaml:"hotplug_size"`
	DiskIOBps         string `json:"disk_io_bps,omitempty" yaml:"disk_io_bps"`
	BandwidthUpload   string `json:"bandwidth_upload,omitempty" yaml:"bandwidth_upload"`
	BandwidthDownload string `json:"bandwidth_download,omitempty" yaml:"bandwidth_download"`
}

type composeRestartSpec struct {
	Policy      string `json:"policy,omitempty" yaml:"policy"`
	Backoff     string `json:"backoff,omitempty" yaml:"backoff"`
	MaxAttempts int    `json:"max_attempts,omitempty" yaml:"max_attempts"`
	StableAfter string `json:"stable_after,omitempty" yaml:"stable_after"`
}

type composeCheckSpec struct {
	HTTP             *composeHTTPCheckSpec `json:"http,omitempty" yaml:"http"`
	TCP              *composeTCPCheckSpec  `json:"tcp,omitempty" yaml:"tcp"`
	Exec             *composeExecCheckSpec `json:"exec,omitempty" yaml:"exec"`
	Type             string                `json:"type,omitempty" yaml:"type"`
	Interval         string                `json:"interval,omitempty" yaml:"interval"`
	Timeout          string                `json:"timeout,omitempty" yaml:"timeout"`
	StartPeriod      string                `json:"start_period,omitempty" yaml:"start_period"`
	FailureThreshold int                   `json:"failure_threshold,omitempty" yaml:"failure_threshold"`
	SuccessThreshold int                   `json:"success_threshold,omitempty" yaml:"success_threshold"`
}

type composeHTTPCheckSpec struct {
	Port           int    `json:"port,omitempty" yaml:"port"`
	Path           string `json:"path,omitempty" yaml:"path"`
	Scheme         string `json:"scheme,omitempty" yaml:"scheme"`
	ExpectedStatus int    `json:"expected_status,omitempty" yaml:"expected_status"`
}

type composeTCPCheckSpec struct {
	Port int `json:"port,omitempty" yaml:"port"`
}

type composeExecCheckSpec struct {
	Command    []string `json:"command,omitempty" yaml:"command"`
	WorkingDir string   `json:"working_dir,omitempty" yaml:"working_dir"`
}

type composeIngressRuleSpec struct {
	Hostname     string `json:"hostname" yaml:"hostname"`
	HostPort     int    `json:"host_port,omitempty" yaml:"host_port"`
	TargetPort   int    `json:"target_port" yaml:"target_port"`
	TLS          bool   `json:"tls,omitempty" yaml:"tls"`
	RedirectHTTP bool   `json:"redirect_http,omitempty" yaml:"redirect_http"`
}

type Plan struct {
	Name    string   `json:"name"`
	File    string   `json:"file"`
	Actions []Action `json:"actions"`
	Summary Summary  `json:"summary"`
}

type Summary struct {
	Create    int `json:"create"`
	Replace   int `json:"replace"`
	Delete    int `json:"delete"`
	Unchanged int `json:"unchanged"`
	Skip      int `json:"skip"`
	Conflict  int `json:"conflict"`
}

type Action struct {
	Action  string `json:"action"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Service string `json:"service,omitempty"`
	Reason  string `json:"reason"`

	instanceID    string
	ingressID     string
	instanceInput map[string]any
	ingressInput  hypeman.IngressNewParams
}

type desiredInstance struct {
	Name    string
	Service string
	Hash    string
	Input   map[string]any
}

type desiredIngress struct {
	Name    string
	Service string
	Hash    string
	Input   hypeman.IngressNewParams
}

func NewRunner(file string, client hypeman.Client, opts ...option.RequestOption) (*Runner, error) {
	spec, err := loadComposeSpec(file)
	if err != nil {
		return nil, err
	}
	return &Runner{
		file:   file,
		spec:   spec,
		client: client,
		opts:   opts,
	}, nil
}

func loadComposeSpec(path string) (composeSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return composeSpec{}, fmt.Errorf("read compose file: %w", err)
	}
	var spec composeSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return composeSpec{}, fmt.Errorf("parse compose file: %w", err)
	}
	if err := validateComposeSpec(&spec); err != nil {
		return composeSpec{}, err
	}
	if err := interpolateComposeSpec(&spec, filepath.Dir(path)); err != nil {
		return composeSpec{}, err
	}
	return spec, nil
}

var composeNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func validateComposeSpec(spec *composeSpec) error {
	if spec.Version != 1 {
		return fmt.Errorf("compose version must be 1")
	}
	if spec.Name == "" {
		return fmt.Errorf("compose name is required")
	}
	if !composeNamePattern.MatchString(spec.Name) {
		return fmt.Errorf("compose name must contain only lowercase letters, digits, and dashes")
	}
	if len(spec.Services) == 0 {
		return fmt.Errorf("compose services must include at least one service")
	}

	for name, service := range spec.Services {
		if !composeNamePattern.MatchString(name) {
			return fmt.Errorf("service %q must contain only lowercase letters, digits, and dashes", name)
		}
		instanceName := composeInstanceName(spec.Name, name)
		if len(instanceName) > 63 {
			return fmt.Errorf("service %q produces instance name %q longer than 63 characters", name, instanceName)
		}
		if service.Image == "" {
			return fmt.Errorf("service %q image is required", name)
		}
		for i, rule := range service.Ingress {
			if rule.Hostname == "" {
				return fmt.Errorf("service %q ingress %d hostname is required", name, i)
			}
			if rule.TargetPort <= 0 {
				return fmt.Errorf("service %q ingress %d target_port must be positive", name, i)
			}
			if rule.HostPort < 0 {
				return fmt.Errorf("service %q ingress %d host_port must be non-negative", name, i)
			}
		}
	}
	return nil
}

var composeInterpolationPattern = regexp.MustCompile(`\$\{(file|env):([^}]+)\}`)

func interpolateComposeSpec(spec *composeSpec, baseDir string) error {
	for serviceName, service := range spec.Services {
		for key, value := range service.Env {
			resolved, err := interpolateComposeValue(value, baseDir)
			if err != nil {
				return fmt.Errorf("service %q env %s: %w", serviceName, key, err)
			}
			service.Env[key] = resolved
		}
		spec.Services[serviceName] = service
	}
	return nil
}

func interpolateComposeValue(value, baseDir string) (string, error) {
	var out strings.Builder
	last := 0
	matches := composeInterpolationPattern.FindAllStringSubmatchIndex(value, -1)
	for _, match := range matches {
		out.WriteString(value[last:match[0]])
		kind := value[match[2]:match[3]]
		arg := strings.TrimSpace(value[match[4]:match[5]])
		switch kind {
		case "env":
			envValue, ok := os.LookupEnv(arg)
			if !ok {
				return "", fmt.Errorf("environment variable %s is not set", arg)
			}
			out.WriteString(envValue)
		case "file":
			path := arg
			if !filepath.IsAbs(path) {
				path = filepath.Join(baseDir, path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read file %s: %w", arg, err)
			}
			out.Write(data)
		}
		last = match[1]
	}
	out.WriteString(value[last:])
	return out.String(), nil
}

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
		var inst hypeman.Instance
		if err := r.client.Post(ctx, "instances", action.instanceInput, &inst, r.opts...); err != nil {
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

func (r *Runner) desiredResources() ([]desiredInstance, []desiredIngress, []string, error) {
	serviceNames := make([]string, 0, len(r.spec.Services))
	imageSet := map[string]struct{}{}
	for name, service := range r.spec.Services {
		serviceNames = append(serviceNames, name)
		imageSet[service.Image] = struct{}{}
	}
	sort.Strings(serviceNames)

	images := make([]string, 0, len(imageSet))
	for image := range imageSet {
		images = append(images, image)
	}
	sort.Strings(images)

	instances := make([]desiredInstance, 0, len(serviceNames))
	var ingresses []desiredIngress
	for _, serviceName := range serviceNames {
		service := r.spec.Services[serviceName]
		instanceName := composeInstanceName(r.spec.Name, serviceName)
		instanceInput := buildComposeInstanceInput(instanceName, service)
		instanceHash, err := shortHash(instanceInput)
		if err != nil {
			return nil, nil, nil, err
		}
		instanceInput["tags"] = composeTags(r.spec.Name, serviceName, composeResourceInstance, instanceHash)
		instances = append(instances, desiredInstance{
			Name:    instanceName,
			Service: serviceName,
			Hash:    instanceHash,
			Input:   instanceInput,
		})

		for i, ingressSpec := range service.Ingress {
			ingressName := composeIngressName(r.spec.Name, serviceName, i)
			ingressInput := buildComposeIngressInput(instanceName, ingressName, ingressSpec)
			ingressHash, err := shortHash(ingressInput)
			if err != nil {
				return nil, nil, nil, err
			}
			ingressInput.Tags = composeTags(r.spec.Name, serviceName, composeResourceIngress, ingressHash)
			ingresses = append(ingresses, desiredIngress{
				Name:    ingressName,
				Service: serviceName,
				Hash:    ingressHash,
				Input:   ingressInput,
			})
		}
	}
	return instances, ingresses, images, nil
}

func buildComposeInstanceInput(instanceName string, service composeServiceSpec) map[string]any {
	input := map[string]any{
		"name":  instanceName,
		"image": service.Image,
	}
	if len(service.Entrypoint) > 0 {
		input["entrypoint"] = service.Entrypoint
	}
	if len(service.Cmd) > 0 {
		input["cmd"] = service.Cmd
	}
	if len(service.Env) > 0 {
		input["env"] = service.Env
	}
	if service.Resources.Vcpus > 0 {
		input["vcpus"] = service.Resources.Vcpus
	}
	if service.Resources.Memory != "" {
		input["size"] = service.Resources.Memory
	}
	if service.Resources.OverlaySize != "" {
		input["overlay_size"] = service.Resources.OverlaySize
	}
	if service.Resources.HotplugSize != "" {
		input["hotplug_size"] = service.Resources.HotplugSize
	}
	if service.Resources.DiskIOBps != "" {
		input["disk_io_bps"] = service.Resources.DiskIOBps
	}
	if service.Resources.BandwidthDownload != "" || service.Resources.BandwidthUpload != "" {
		network := map[string]any{}
		if service.Resources.BandwidthDownload != "" {
			network["bandwidth_download"] = service.Resources.BandwidthDownload
		}
		if service.Resources.BandwidthUpload != "" {
			network["bandwidth_upload"] = service.Resources.BandwidthUpload
		}
		input["network"] = network
	}
	if service.Restart != nil {
		input["restart_policy"] = buildComposeRestartPayload(service.Restart)
	}
	if service.Health != nil {
		input["health_check"] = service.Health
	}
	return input
}

func buildComposeRestartPayload(restart *composeRestartSpec) map[string]any {
	payload := map[string]any{}
	if restart.Policy != "" {
		payload["policy"] = strings.ReplaceAll(restart.Policy, "-", "_")
	}
	if restart.Backoff != "" {
		payload["backoff"] = restart.Backoff
	}
	if restart.MaxAttempts > 0 {
		payload["max_attempts"] = restart.MaxAttempts
	}
	if restart.StableAfter != "" {
		payload["stable_after"] = restart.StableAfter
	}
	return payload
}

func buildComposeIngressInput(instanceName, ingressName string, spec composeIngressRuleSpec) hypeman.IngressNewParams {
	hostPort := spec.HostPort
	if hostPort == 0 {
		hostPort = 80
	}
	return hypeman.IngressNewParams{
		Name: ingressName,
		Rules: []hypeman.IngressRuleParam{
			{
				Match: hypeman.IngressMatchParam{
					Hostname: spec.Hostname,
					Port:     hypeman.Int(int64(hostPort)),
				},
				Target: hypeman.IngressTargetParam{
					Instance: instanceName,
					Port:     int64(spec.TargetPort),
				},
				Tls:          hypeman.Bool(spec.TLS),
				RedirectHTTP: hypeman.Bool(spec.RedirectHTTP),
			},
		},
	}
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

func composeInstanceName(composeName, serviceName string) string {
	return composeName + "-" + serviceName
}

func composeIngressName(composeName, serviceName string, index int) string {
	return fmt.Sprintf("%s-%s-%d", composeName, serviceName, index)
}

func composeTags(composeName, serviceName, resource, hash string) map[string]string {
	return map[string]string{
		composeTagName:     composeName,
		composeTagService:  serviceName,
		composeTagResource: resource,
		composeTagHash:     hash,
	}
}

func shortHash(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12], nil
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
