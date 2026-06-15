package compose

import (
	"strings"

	"github.com/kernel/hypeman-go"
)

// HealthCheckInput is a neutral, plain-value description of a workload health
// check. It is shared by compose, the imperative run command, and the update
// subcommands so the param-construction logic does not drift.
type HealthCheckInput struct {
	Type             string
	Interval         string
	Timeout          string
	StartPeriod      string
	FailureThreshold int64
	SuccessThreshold int64
	HTTP             *HealthCheckHTTPInput
	TCP              *HealthCheckTCPInput
	Exec             *HealthCheckExecInput
}

type HealthCheckHTTPInput struct {
	Port           int64
	Path           string
	Scheme         string
	ExpectedStatus int64
}

type HealthCheckTCPInput struct {
	Port int64
}

type HealthCheckExecInput struct {
	Command    []string
	WorkingDir string
}

// RestartPolicyInput is a neutral, plain-value description of a restart policy.
type RestartPolicyInput struct {
	Policy      string
	Backoff     string
	MaxAttempts int64
	StableAfter string
}

func BuildHealthCheckParam(in HealthCheckInput) hypeman.HealthCheckParam {
	health := hypeman.HealthCheckParam{}
	if in.Type != "" {
		health.Type = hypeman.HealthCheckType(strings.ToLower(in.Type))
	}
	if in.HTTP != nil {
		health.Type = defaultHealthCheckType(health.Type, hypeman.HealthCheckTypeHTTP)
		health.HTTP = hypeman.HealthCheckHTTPParam{
			Port: in.HTTP.Port,
		}
		if in.HTTP.Path != "" {
			health.HTTP.Path = hypeman.String(in.HTTP.Path)
		}
		if in.HTTP.Scheme != "" {
			health.HTTP.Scheme = hypeman.HealthCheckHTTPScheme(strings.ToLower(in.HTTP.Scheme))
		}
		if in.HTTP.ExpectedStatus > 0 {
			health.HTTP.ExpectedStatus = hypeman.Int(in.HTTP.ExpectedStatus)
		}
	}
	if in.TCP != nil {
		health.Type = defaultHealthCheckType(health.Type, hypeman.HealthCheckTypeTcp)
		health.Tcp = hypeman.HealthCheckTcpParam{Port: in.TCP.Port}
	}
	if in.Exec != nil {
		health.Type = defaultHealthCheckType(health.Type, hypeman.HealthCheckTypeExec)
		health.Exec = hypeman.HealthCheckExecParam{
			Command: in.Exec.Command,
		}
		if in.Exec.WorkingDir != "" {
			health.Exec.WorkingDir = hypeman.String(in.Exec.WorkingDir)
		}
	}
	if in.Interval != "" {
		health.Interval = hypeman.String(in.Interval)
	}
	if in.Timeout != "" {
		health.Timeout = hypeman.String(in.Timeout)
	}
	if in.StartPeriod != "" {
		health.StartPeriod = hypeman.String(in.StartPeriod)
	}
	if in.FailureThreshold > 0 {
		health.FailureThreshold = hypeman.Int(in.FailureThreshold)
	}
	if in.SuccessThreshold > 0 {
		health.SuccessThreshold = hypeman.Int(in.SuccessThreshold)
	}
	return health
}

func BuildRestartPolicyParam(in RestartPolicyInput) hypeman.RestartPolicyParam {
	policy := hypeman.RestartPolicyParam{}
	if in.Policy != "" {
		policy.Policy = hypeman.RestartPolicyPolicy(strings.ReplaceAll(in.Policy, "-", "_"))
	}
	if in.Backoff != "" {
		policy.Backoff = hypeman.String(in.Backoff)
	}
	if in.MaxAttempts > 0 {
		policy.MaxAttempts = hypeman.Int(in.MaxAttempts)
	}
	if in.StableAfter != "" {
		policy.StableAfter = hypeman.String(in.StableAfter)
	}
	return policy
}

func defaultHealthCheckType(current, fallback hypeman.HealthCheckType) hypeman.HealthCheckType {
	if current != "" {
		return current
	}
	return fallback
}
