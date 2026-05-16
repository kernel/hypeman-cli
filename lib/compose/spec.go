package compose

import (
	"fmt"
	"os"
	"path/filepath"
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
