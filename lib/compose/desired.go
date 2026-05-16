package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kernel/hypeman-go"
)

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
