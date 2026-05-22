package compose

import (
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
	if err := yaml.Unmarshal(data, &spec); err != nil {
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
