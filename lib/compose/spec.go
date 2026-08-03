package compose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type composeSpec struct {
	Version  int                           `json:"version" yaml:"version"`
	Name     string                        `json:"name" yaml:"name"`
	Services map[string]composeServiceSpec `json:"services" yaml:"services"`
	Volumes  map[string]composeVolumeSpec  `json:"volumes,omitempty" yaml:"volumes,omitempty"`
}

// composeVolumeSpec declares a retained named volume. Volumes are created
// before instances, survive instance replacement and `compose down`, and are
// only destroyed by an explicit destructive option (`compose down --volumes`).
type composeVolumeSpec struct {
	Name   string `json:"name,omitempty" yaml:"name"`
	SizeGB int64  `json:"size_gb" yaml:"size_gb"`
}

// composeVolumeMountSpec attaches a declared volume to a service. It accepts
// either the shorthand string form "volume:/abs/path[:ro|rw]" or the mapping
// form:
//
//	volume: data
//	mount_path: /var/lib/data
//	readonly: true
type composeVolumeMountSpec struct {
	Volume    string `json:"volume" yaml:"volume"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
	Readonly  bool   `json:"readonly,omitempty" yaml:"readonly"`
}

func (m *composeVolumeMountSpec) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var shorthand string
		if err := node.Decode(&shorthand); err != nil {
			return err
		}
		parsed, err := parseComposeVolumeMountShorthand(shorthand)
		if err != nil {
			return fmt.Errorf("line %d: %w", node.Line, err)
		}
		*m = parsed
		return nil
	case yaml.MappingNode:
		// The top-level strict decoder hands custom unmarshalers the raw
		// node, so enforce known fields here explicitly.
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			switch key.Value {
			case "volume", "mount_path", "readonly":
			default:
				return fmt.Errorf("line %d: field %s not found in volume mount", key.Line, key.Value)
			}
		}
		type rawMount composeVolumeMountSpec
		var raw rawMount
		if err := node.Decode(&raw); err != nil {
			return err
		}
		*m = composeVolumeMountSpec(raw)
		return nil
	default:
		return fmt.Errorf("line %d: volume mount must be a string or mapping", node.Line)
	}
}

func parseComposeVolumeMountShorthand(value string) (composeVolumeMountSpec, error) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return composeVolumeMountSpec{}, fmt.Errorf("volume mount %q must be in the form volume:/abs/path[:ro|rw]", value)
	}
	mount := composeVolumeMountSpec{
		Volume:    strings.TrimSpace(parts[0]),
		MountPath: parts[1],
	}
	if mount.Volume == "" {
		return composeVolumeMountSpec{}, fmt.Errorf("volume mount %q is missing the volume name", value)
	}
	if mount.MountPath == "" {
		return composeVolumeMountSpec{}, fmt.Errorf("volume mount %q is missing the mount path", value)
	}
	if len(parts) == 3 {
		switch parts[2] {
		case "ro":
			mount.Readonly = true
		case "rw":
			mount.Readonly = false
		default:
			return composeVolumeMountSpec{}, fmt.Errorf("volume mount %q has invalid mode %q (must be ro or rw)", value, parts[2])
		}
	}
	return mount, nil
}

type composeServiceSpec struct {
	Name       string                   `json:"name,omitempty" yaml:"name"`
	Image      string                   `json:"image" yaml:"image"`
	Dockerfile string                   `json:"dockerfile,omitempty" yaml:"dockerfile"`
	Entrypoint []string                 `json:"entrypoint,omitempty" yaml:"entrypoint"`
	Cmd        []string                 `json:"cmd,omitempty" yaml:"cmd"`
	Env        map[string]string        `json:"env,omitempty" yaml:"env"`
	Resources  composeResourcesSpec     `json:"resources,omitempty" yaml:"resources"`
	Restart    *composeRestartSpec      `json:"restart,omitempty" yaml:"restart"`
	Health     *composeCheckSpec        `json:"healthcheck,omitempty" yaml:"healthcheck"`
	Ingress    []composeIngressRuleSpec `json:"ingress,omitempty" yaml:"ingress"`
	Volumes    []composeVolumeMountSpec `json:"volumes,omitempty" yaml:"volumes"`
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
	Name         string `json:"name,omitempty" yaml:"name"`
	Hostname     string `json:"hostname" yaml:"hostname"`
	HostPort     int    `json:"host_port,omitempty" yaml:"host_port"`
	TargetPort   int    `json:"target_port" yaml:"target_port"`
	TLS          bool   `json:"tls,omitempty" yaml:"tls"`
	RedirectHTTP bool   `json:"redirect_http,omitempty" yaml:"redirect_http"`
}

func loadComposeSpec(path string) (composeSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return composeSpec{}, fmt.Errorf("read compose file: %w", err)
	}
	var spec composeSpec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return composeSpec{}, fmt.Errorf("parse compose file: %w", err)
	}
	if err := interpolateComposeSpec(&spec, filepath.Dir(path)); err != nil {
		return composeSpec{}, err
	}
	if err := validateComposeSpec(&spec); err != nil {
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

	volumeNames := map[string]string{}
	for key, volume := range spec.Volumes {
		if !composeNamePattern.MatchString(key) {
			return fmt.Errorf("volume %q must contain only lowercase letters, digits, and dashes", key)
		}
		if volume.Name != "" && !composeNamePattern.MatchString(volume.Name) {
			return fmt.Errorf("volume %q name must contain only lowercase letters, digits, and dashes", key)
		}
		volumeName := composeVolumeName(spec.Name, key, volume)
		if len(volumeName) > 63 {
			return fmt.Errorf("volume %q produces volume name %q longer than 63 characters", key, volumeName)
		}
		if existing, ok := volumeNames[volumeName]; ok {
			return fmt.Errorf("volume %q produces duplicate volume name %q already used by volume %q", key, volumeName, existing)
		}
		volumeNames[volumeName] = key
		if volume.SizeGB <= 0 {
			return fmt.Errorf("volume %q size_gb must be positive", key)
		}
	}

	instanceNames := map[string]string{}
	ingressNames := map[string]string{}
	for name, service := range spec.Services {
		if !composeNamePattern.MatchString(name) {
			return fmt.Errorf("service %q must contain only lowercase letters, digits, and dashes", name)
		}
		instanceName := composeInstanceName(spec.Name, name, service)
		if service.Name != "" && !composeNamePattern.MatchString(service.Name) {
			return fmt.Errorf("service %q name must contain only lowercase letters, digits, and dashes", name)
		}
		if len(instanceName) > 63 {
			return fmt.Errorf("service %q produces instance name %q longer than 63 characters", name, instanceName)
		}
		if existing, ok := instanceNames[instanceName]; ok {
			return fmt.Errorf("service %q produces duplicate instance name %q already used by service %q", name, instanceName, existing)
		}
		instanceNames[instanceName] = name
		if service.Image == "" && service.Dockerfile == "" {
			return fmt.Errorf("service %q image or dockerfile is required", name)
		}
		if service.Image != "" && service.Dockerfile != "" {
			return fmt.Errorf("service %q cannot include both image and dockerfile", name)
		}
		mountPaths := map[string]int{}
		mountedVolumes := map[string]int{}
		for i, mount := range service.Volumes {
			if mount.Volume == "" {
				return fmt.Errorf("service %q volume mount %d volume is required", name, i)
			}
			if _, ok := spec.Volumes[mount.Volume]; !ok {
				return fmt.Errorf("service %q volume mount %d references unknown volume %q", name, i, mount.Volume)
			}
			if mount.MountPath == "" {
				return fmt.Errorf("service %q volume mount %d mount_path is required", name, i)
			}
			if !strings.HasPrefix(mount.MountPath, "/") {
				return fmt.Errorf("service %q volume mount %d mount_path %q must be an absolute path", name, i, mount.MountPath)
			}
			if existing, ok := mountPaths[mount.MountPath]; ok {
				return fmt.Errorf("service %q volume mount %d duplicates mount_path %q already used by mount %d", name, i, mount.MountPath, existing)
			}
			mountPaths[mount.MountPath] = i
			if existing, ok := mountedVolumes[mount.Volume]; ok {
				return fmt.Errorf("service %q volume mount %d mounts volume %q more than once (already used by mount %d)", name, i, mount.Volume, existing)
			}
			mountedVolumes[mount.Volume] = i
		}
		for i, rule := range service.Ingress {
			ingressName := composeIngressName(spec.Name, name, i, rule)
			if rule.Name != "" && !composeNamePattern.MatchString(rule.Name) {
				return fmt.Errorf("service %q ingress %d name must contain only lowercase letters, digits, and dashes", name, i)
			}
			if existing, ok := ingressNames[ingressName]; ok {
				return fmt.Errorf("service %q ingress %d produces duplicate ingress name %q already used by %s", name, i, ingressName, existing)
			}
			ingressNames[ingressName] = fmt.Sprintf("service %q ingress %d", name, i)
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
	return interpolateComposeFields(reflect.ValueOf(spec).Elem(), baseDir, "compose")
}

func interpolateComposeFields(value reflect.Value, baseDir, path string) error {
	if !value.IsValid() {
		return nil
	}
	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return interpolateComposeFields(value.Elem(), baseDir, path)
	case reflect.String:
		resolved, err := interpolateComposeValue(value.String(), baseDir)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		value.SetString(resolved)
	case reflect.Struct:
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := composeYAMLFieldName(field)
			if name == "" {
				continue
			}
			if err := interpolateComposeFields(value.Field(i), baseDir, path+"."+name); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := interpolateComposeFields(value.Index(i), baseDir, fmt.Sprintf("%s.%d", path, i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			item := reflect.New(value.Type().Elem()).Elem()
			item.Set(value.MapIndex(key))
			if err := interpolateComposeFields(item, baseDir, path+"."+fmt.Sprint(key.Interface())); err != nil {
				return err
			}
			value.SetMapIndex(key, item)
		}
	}
	return nil
}

func composeYAMLFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	if name != "" {
		return name
	}
	return field.Name
}

func interpolateComposeValue(value, baseDir string) (string, error) {
	return interpolateComposeValueDepth(value, baseDir, 0)
}

func interpolateComposeValueDepth(value, baseDir string, depth int) (string, error) {
	if depth > 16 {
		return "", fmt.Errorf("interpolation depth exceeded")
	}
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
			rendered, err := interpolateComposeValueDepth(string(data), baseDir, depth+1)
			if err != nil {
				return "", fmt.Errorf("render file %s: %w", arg, err)
			}
			out.WriteString(rendered)
		}
		last = match[1]
	}
	out.WriteString(value[last:])
	return out.String(), nil
}
