package cmd

import (
	"fmt"

	"github.com/kernel/hypeman-cli/lib/compose"
	"github.com/urfave/cli/v3"
)

func healthCheckFlags(prefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  prefix + "type",
			Usage: `Health probe type: "none", "http", "tcp", or "exec"`,
		},
		&cli.StringFlag{
			Name:  prefix + "interval",
			Usage: `Delay between checks (e.g., "10s")`,
		},
		&cli.StringFlag{
			Name:  prefix + "timeout",
			Usage: `Per-check timeout (e.g., "2s")`,
		},
		&cli.StringFlag{
			Name:  prefix + "start-period",
			Usage: `Startup grace period before failures count (e.g., "30s")`,
		},
		&cli.IntFlag{
			Name:  prefix + "failure-threshold",
			Usage: "Consecutive failed checks required to mark the workload unhealthy",
		},
		&cli.IntFlag{
			Name:  prefix + "success-threshold",
			Usage: "Consecutive successful checks required to mark the workload healthy",
		},
		&cli.IntFlag{
			Name:  prefix + "http-port",
			Usage: "Port to probe for an HTTP health check",
		},
		&cli.StringFlag{
			Name:  prefix + "http-path",
			Usage: "HTTP path to request for an HTTP health check",
		},
		&cli.StringFlag{
			Name:  prefix + "http-scheme",
			Usage: `HTTP scheme for an HTTP health check: "http" or "https"`,
		},
		&cli.IntFlag{
			Name:  prefix + "http-expected-status",
			Usage: "Exact status code required for a successful HTTP probe",
		},
		&cli.IntFlag{
			Name:  prefix + "tcp-port",
			Usage: "Port to open for a TCP health check",
		},
		&cli.StringSliceFlag{
			Name:  prefix + "exec",
			Usage: "Command and arguments for an exec health check (can be repeated)",
		},
		&cli.StringFlag{
			Name:  prefix + "exec-working-dir",
			Usage: "Working directory for an exec health check",
		},
	}
}

func restartPolicyFlags(prefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  prefix + "policy",
			Usage: `Restart behavior: "never", "always", or "on_failure"`,
		},
		&cli.StringFlag{
			Name:  prefix + "backoff",
			Usage: `Delay before each restart attempt (e.g., "5s")`,
		},
		&cli.IntFlag{
			Name:  prefix + "max-attempts",
			Usage: "Consecutive restart attempts before blocking retries (0 means unlimited)",
		},
		&cli.StringFlag{
			Name:  prefix + "stable-after",
			Usage: `Running this long resets the consecutive restart attempt count (e.g., "10m")`,
		},
	}
}

func parseHealthCheckInput(cmd *cli.Command, prefix string) (compose.HealthCheckInput, bool, error) {
	typeFlag := prefix + "type"
	intervalFlag := prefix + "interval"
	timeoutFlag := prefix + "timeout"
	startPeriodFlag := prefix + "start-period"
	failureThresholdFlag := prefix + "failure-threshold"
	successThresholdFlag := prefix + "success-threshold"
	httpPortFlag := prefix + "http-port"
	httpPathFlag := prefix + "http-path"
	httpSchemeFlag := prefix + "http-scheme"
	httpExpectedStatusFlag := prefix + "http-expected-status"
	tcpPortFlag := prefix + "tcp-port"
	execFlag := prefix + "exec"
	execWorkingDirFlag := prefix + "exec-working-dir"

	// A probe sub-block is engaged by its required-field flag (http-port, tcp-port,
	// exec command); those fields are api:"required", so a secondary flag alone
	// (--http-path/-scheme/-expected-status or --exec-working-dir) would build a probe
	// with a zero/empty required value. Reject that explicitly.
	httpSet := cmd.IsSet(httpPortFlag) || cmd.IsSet(httpPathFlag) || cmd.IsSet(httpSchemeFlag) || cmd.IsSet(httpExpectedStatusFlag)
	if httpSet && !cmd.IsSet(httpPortFlag) {
		return compose.HealthCheckInput{}, false, fmt.Errorf("--%shttp-port is required when configuring an HTTP health check", prefix)
	}
	if cmd.IsSet(execWorkingDirFlag) && !cmd.IsSet(execFlag) {
		return compose.HealthCheckInput{}, false, fmt.Errorf("--%sexec is required when setting --%sexec-working-dir", prefix, prefix)
	}
	tcpSet := cmd.IsSet(tcpPortFlag)
	execSet := cmd.IsSet(execFlag)

	set := cmd.IsSet(typeFlag) || cmd.IsSet(intervalFlag) || cmd.IsSet(timeoutFlag) ||
		cmd.IsSet(startPeriodFlag) || cmd.IsSet(failureThresholdFlag) || cmd.IsSet(successThresholdFlag) ||
		httpSet || tcpSet || execSet
	if !set {
		return compose.HealthCheckInput{}, false, nil
	}

	in := compose.HealthCheckInput{
		Type:             cmd.String(typeFlag),
		Interval:         cmd.String(intervalFlag),
		Timeout:          cmd.String(timeoutFlag),
		StartPeriod:      cmd.String(startPeriodFlag),
		FailureThreshold: int64(cmd.Int(failureThresholdFlag)),
		SuccessThreshold: int64(cmd.Int(successThresholdFlag)),
	}
	if httpSet {
		in.HTTP = &compose.HealthCheckHTTPInput{
			Port:           int64(cmd.Int(httpPortFlag)),
			Path:           cmd.String(httpPathFlag),
			Scheme:         cmd.String(httpSchemeFlag),
			ExpectedStatus: int64(cmd.Int(httpExpectedStatusFlag)),
		}
	}
	if tcpSet {
		in.TCP = &compose.HealthCheckTCPInput{Port: int64(cmd.Int(tcpPortFlag))}
	}
	if execSet {
		in.Exec = &compose.HealthCheckExecInput{
			Command:    cmd.StringSlice(execFlag),
			WorkingDir: cmd.String(execWorkingDirFlag),
		}
	}
	return in, true, nil
}

func parseRestartPolicyInput(cmd *cli.Command, prefix string) (compose.RestartPolicyInput, bool) {
	policyFlag := prefix + "policy"
	backoffFlag := prefix + "backoff"
	maxAttemptsFlag := prefix + "max-attempts"
	stableAfterFlag := prefix + "stable-after"

	set := cmd.IsSet(policyFlag) || cmd.IsSet(backoffFlag) || cmd.IsSet(maxAttemptsFlag) || cmd.IsSet(stableAfterFlag)
	if !set {
		return compose.RestartPolicyInput{}, false
	}

	in := compose.RestartPolicyInput{
		Policy:      cmd.String(policyFlag),
		Backoff:     cmd.String(backoffFlag),
		StableAfter: cmd.String(stableAfterFlag),
	}
	// Send max_attempts only when explicitly provided, so a PATCH with --max-attempts 0
	// clears the limit to unlimited rather than being omitted (a no-op) by omitzero.
	if cmd.IsSet(maxAttemptsFlag) {
		v := int64(cmd.Int(maxAttemptsFlag))
		in.MaxAttempts = &v
	}
	return in, true
}
