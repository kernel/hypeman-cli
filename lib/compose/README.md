# Command Features

## Compose

`hypeman compose` is a lightweight way to declare a small workload for Hypeman from images or Dockerfiles.


```yaml
version: 1
name: hypeship-otel

services:
  otelcol:
    image: otel/opentelemetry-collector-contrib:0.108.0
    cmd: ["--config=env:OTELCOL_CONFIG"]
    env:
      OTELCOL_CONFIG: ${file:otelcol.yaml}
      SIGNOZ_ACCESS_TOKEN: ${env:SIGNOZ_ACCESS_TOKEN}
    resources:
      vcpus: 8
      memory: 4GB
    restart:
      policy: on_failure
      backoff: 5s
      max_attempts: 10
    healthcheck:
      http:
        port: 13133
        path: /
      interval: 10s
      timeout: 2s
      failure_threshold: 3
    ingress:
      - hostname: otel.example.com
        host_port: 443
        target_port: 4318
        tls: true
```

### Commands

Preview the changes:

```sh
hypeman compose plan -f hypeman.compose.yaml
```

Apply the file:

```sh
hypeman compose up -f hypeman.compose.yaml
```

Delete resources owned by the file:

```sh
hypeman compose down -f hypeman.compose.yaml
```

`up` waits for newly created instances to reach `Running` by default. Use `--wait=false` to skip that wait, or `--wait-timeout 30s` to change the per-instance timeout.

If a managed instance or ingress exists but the rendered spec changed, `up` reports that replacement is required and exits without changing resources. Re-run with `--replace` to recreate changed resources.

All compose commands honor global output flags such as `--format json`, `--format yaml`, and `--transform`.

### How It Works

`plan` renders the desired resources from the compose file, checks whether referenced images exist, then compares the desired instances and ingresses against existing resources.

`up` applies the plan in order:

1. build Dockerfile services whose generated images are missing
2. ensure referenced images exist and are ready
3. create or replace instances
4. create or replace ingresses

`down` deletes only instances and ingresses tagged as owned by the compose file. Images are left in place because they can be shared by normal `hypeman run` usage or other compose files.

Instances and ingresses get compose ownership tags:

```text
hypeman.compose.name
hypeman.compose.service
hypeman.compose.resource
hypeman.compose.hash
```

The hash is computed from the rendered resource spec before ownership tags are added. Re-running the same file is idempotent: matching resources are reported as unchanged, changed managed resources require `--replace`, and unmanaged resources with the same name are reported as conflicts.

### Interpolation

String values can embed local files or environment variables:

```yaml
ingress:
  - hostname: ${env:OTEL_COLLECTOR_VM_HOSTNAME}
    target_port: 4318

env:
  OTELCOL_CONFIG: ${file:otelcol.yaml}
  SIGNOZ_ACCESS_TOKEN: ${env:SIGNOZ_ACCESS_TOKEN}
```

File paths are resolved relative to the compose file. Loaded file contents are rendered the same way, so an `otelcol.yaml` referenced with `${file:otelcol.yaml}` can contain `${env:OTEL_COLLECTOR_VM_TOKEN}` or another `${file:...}` reference. Missing files or environment variables fail before any resources are applied.

### Dockerfile Services

A service can use `dockerfile` instead of `image`:

```yaml
services:
  worker:
    dockerfile: ./Dockerfile
    cmd: ["./worker"]
    env:
      CONFIG: ${file:worker.yaml}
    restart:
      policy: on_failure
```

The Dockerfile path is resolved relative to the compose file. The build context is the directory containing that Dockerfile. `compose up` creates a source archive, starts a Hypeman build, waits for the generated image to become ready, then creates the instance from that image.

Compose generates the build image name from the compose name, service name, Dockerfile, and build context hash. Re-running the same file reuses the existing image; changing the Dockerfile or context produces a new image name and makes the managed instance require replacement.

`image` and `dockerfile` are mutually exclusive for now. Use `image` for off-the-shelf images and `dockerfile` for Hypeman-built images.

### OTel Collector Example

The OTel collector can run from the upstream collector image without rebuilding it. Put the collector config in `otelcol.yaml`, reference it with `${file:otelcol.yaml}`, and pass `--config=env:OTELCOL_CONFIG` as the service command. Restart policy and healthcheck settings are applied to the instance create request, while ingress exposes only the collector port you choose.
