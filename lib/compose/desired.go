package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kernel/hypeman-go"
)

type desiredInstance struct {
	Name    string
	Service string
	Hash    string
	Input   hypeman.InstanceNewParams
}

type desiredIngress struct {
	Name    string
	Service string
	Hash    string
	Input   hypeman.IngressNewParams
}

type desiredVolume struct {
	Name  string
	Hash  string
	Input hypeman.VolumeNewParams
}

func (r *Runner) desiredResources() ([]desiredBuild, []desiredVolume, []desiredInstance, []desiredIngress, []string, error) {
	serviceNames := make([]string, 0, len(r.spec.Services))
	imageSet := map[string]struct{}{}
	for name, service := range r.spec.Services {
		serviceNames = append(serviceNames, name)
		if service.Image != "" {
			imageSet[service.Image] = struct{}{}
		}
	}
	sort.Strings(serviceNames)

	images := make([]string, 0, len(imageSet))
	for image := range imageSet {
		images = append(images, image)
	}
	sort.Strings(images)

	volumeKeys := make([]string, 0, len(r.spec.Volumes))
	for key := range r.spec.Volumes {
		volumeKeys = append(volumeKeys, key)
	}
	sort.Strings(volumeKeys)
	volumeNames := make(map[string]string, len(volumeKeys))
	volumes := make([]desiredVolume, 0, len(volumeKeys))
	for _, key := range volumeKeys {
		volumeSpec := r.spec.Volumes[key]
		volumeName := composeVolumeName(r.spec.Name, key, volumeSpec)
		volumeNames[key] = volumeName
		volumeInput := hypeman.VolumeNewParams{
			Name:   volumeName,
			SizeGB: volumeSpec.SizeGB,
		}
		volumeHash, err := shortHash(volumeInput)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		volumeInput.Tags = composeVolumeTags(r.spec.Name, volumeHash)
		volumes = append(volumes, desiredVolume{
			Name:  volumeName,
			Hash:  volumeHash,
			Input: volumeInput,
		})
	}

	var builds []desiredBuild
	instances := make([]desiredInstance, 0, len(serviceNames))
	var ingresses []desiredIngress
	for _, serviceName := range serviceNames {
		service := r.spec.Services[serviceName]
		if service.Dockerfile != "" {
			build, err := r.desiredBuildForService(serviceName, service)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			builds = append(builds, build)
			service.Image = build.Image
		}
		instanceName := composeInstanceName(r.spec.Name, serviceName, service)
		instanceInput := buildComposeInstanceInput(instanceName, service, volumeNames)
		instanceHash, err := shortHash(instanceInput)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		instanceInput.Tags = composeTags(r.spec.Name, serviceName, composeResourceInstance, instanceHash)
		instances = append(instances, desiredInstance{
			Name:    instanceName,
			Service: serviceName,
			Hash:    instanceHash,
			Input:   instanceInput,
		})

		for i, ingressSpec := range service.Ingress {
			ingressName := composeIngressName(r.spec.Name, serviceName, i, ingressSpec)
			ingressInput := buildComposeIngressInput(instanceName, ingressName, ingressSpec)
			ingressHash, err := shortHash(ingressInput)
			if err != nil {
				return nil, nil, nil, nil, nil, err
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
	return builds, volumes, instances, ingresses, images, nil
}

// buildComposeInstanceInput renders the instance create params for a service.
// volumeNames maps compose volume keys to their resolved Hypeman volume names.
// Mounts carry the volume *name* (not the server-assigned ID) so the rendered
// hash stays stable across volume creation; names are resolved to IDs at apply
// time, immediately before instance creation.
func buildComposeInstanceInput(instanceName string, service composeServiceSpec, volumeNames map[string]string) hypeman.InstanceNewParams {
	input := hypeman.InstanceNewParams{
		Name:  instanceName,
		Image: service.Image,
	}
	if len(service.Volumes) > 0 {
		mounts := make([]hypeman.VolumeMountParam, 0, len(service.Volumes))
		for _, mount := range service.Volumes {
			volumeMount := hypeman.VolumeMountParam{
				VolumeID:  volumeNames[mount.Volume],
				MountPath: mount.MountPath,
			}
			if mount.Readonly {
				volumeMount.Readonly = hypeman.Bool(true)
			}
			mounts = append(mounts, volumeMount)
		}
		input.Volumes = mounts
	}
	if len(service.Entrypoint) > 0 {
		input.Entrypoint = service.Entrypoint
	}
	if len(service.Cmd) > 0 {
		input.Cmd = service.Cmd
	}
	if len(service.Env) > 0 {
		input.Env = service.Env
	}
	if service.Resources.Vcpus > 0 {
		input.Vcpus = hypeman.Int(int64(service.Resources.Vcpus))
	}
	if service.Resources.Memory != "" {
		input.Size = hypeman.String(service.Resources.Memory)
	}
	if service.Resources.OverlaySize != "" {
		input.OverlaySize = hypeman.String(service.Resources.OverlaySize)
	}
	if service.Resources.HotplugSize != "" {
		input.HotplugSize = hypeman.String(service.Resources.HotplugSize)
	}
	if service.Resources.DiskIOBps != "" {
		input.DiskIoBps = hypeman.String(service.Resources.DiskIOBps)
	}
	if service.Resources.BandwidthDownload != "" || service.Resources.BandwidthUpload != "" {
		if service.Resources.BandwidthDownload != "" {
			input.Network.BandwidthDownload = hypeman.String(service.Resources.BandwidthDownload)
		}
		if service.Resources.BandwidthUpload != "" {
			input.Network.BandwidthUpload = hypeman.String(service.Resources.BandwidthUpload)
		}
	}
	if service.Restart != nil {
		input.RestartPolicy = buildComposeRestartPolicy(service.Restart)
	}
	if service.Health != nil {
		input.HealthCheck = buildComposeHealthCheck(service.Health)
	}
	return input
}

func updateDesiredInstanceImage(instances []desiredInstance, composeName, serviceName, image string) error {
	if image == "" {
		return nil
	}
	for i := range instances {
		if instances[i].Service != serviceName {
			continue
		}
		instances[i].Input.Image = image
		instances[i].Input.Tags = nil
		hash, err := shortHash(instances[i].Input)
		if err != nil {
			return err
		}
		instances[i].Hash = hash
		instances[i].Input.Tags = composeTags(composeName, serviceName, composeResourceInstance, hash)
	}
	return nil
}

func buildComposeRestartPolicy(restart *composeRestartSpec) hypeman.RestartPolicyParam {
	in := RestartPolicyInput{
		Policy:      restart.Policy,
		Backoff:     restart.Backoff,
		StableAfter: restart.StableAfter,
	}
	if restart.MaxAttempts > 0 {
		v := int64(restart.MaxAttempts)
		in.MaxAttempts = &v
	}
	return BuildRestartPolicyParam(in)
}

func buildComposeHealthCheck(check *composeCheckSpec) hypeman.HealthCheckParam {
	in := HealthCheckInput{
		Type:             check.Type,
		Interval:         check.Interval,
		Timeout:          check.Timeout,
		StartPeriod:      check.StartPeriod,
		FailureThreshold: int64(check.FailureThreshold),
		SuccessThreshold: int64(check.SuccessThreshold),
	}
	if check.HTTP != nil {
		in.HTTP = &HealthCheckHTTPInput{
			Port:           int64(check.HTTP.Port),
			Path:           check.HTTP.Path,
			Scheme:         check.HTTP.Scheme,
			ExpectedStatus: int64(check.HTTP.ExpectedStatus),
		}
	}
	if check.TCP != nil {
		in.TCP = &HealthCheckTCPInput{Port: int64(check.TCP.Port)}
	}
	if check.Exec != nil {
		in.Exec = &HealthCheckExecInput{
			Command:    check.Exec.Command,
			WorkingDir: check.Exec.WorkingDir,
		}
	}
	return BuildHealthCheckParam(in)
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

func composeInstanceName(composeName, serviceName string, service composeServiceSpec) string {
	if service.Name != "" {
		return service.Name
	}
	return composeName + "-" + serviceName
}

func composeIngressName(composeName, serviceName string, index int, ingress composeIngressRuleSpec) string {
	if ingress.Name != "" {
		return ingress.Name
	}
	return fmt.Sprintf("%s-%s-%d", composeName, serviceName, index)
}

func composeVolumeName(composeName, key string, volume composeVolumeSpec) string {
	if volume.Name != "" {
		return volume.Name
	}
	return composeName + "-" + key
}

func composeTags(composeName, serviceName, resource, hash string) map[string]string {
	return map[string]string{
		composeTagName:     composeName,
		composeTagService:  serviceName,
		composeTagResource: resource,
		composeTagHash:     hash,
	}
}

// composeVolumeTags tags a retained volume with compose ownership. Volumes are
// shared across services, so they carry no service tag.
func composeVolumeTags(composeName, hash string) map[string]string {
	return map[string]string{
		composeTagName:     composeName,
		composeTagResource: composeResourceVolume,
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
