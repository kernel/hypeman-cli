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

By default, compose names resources from the compose name and service key:

```text
instance: <compose name>-<service>
ingress:  <compose name>-<service>-<index>
```

Set `name` on a service or ingress rule when a stable external name is required:

```yaml
version: 1
name: hypeship

services:
  otelcol:
    name: hypeship-otelcol-${env:DEPLOY_ENV}
    image: otel/opentelemetry-collector-contrib:0.108.0
    ingress:
      - name: hypeship-otelcol-${env:DEPLOY_ENV}-otlp
        hostname: hypeship-otelcol-${env:DEPLOY_ENV}.dev-yul-hypeman-0.kernel.sh
        host_port: 443
        target_port: 3000
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

Retained volumes are never deleted by `up` or `down`. Passing `--volumes` to `down` also deletes the volumes owned by the file and **destroys their data**:

```sh
hypeman compose down -f hypeman.compose.yaml --volumes
```

All compose commands honor global output flags such as `--format json`, `--format yaml`, and `--transform`.

### How It Works

`plan` renders the desired resources from the compose file, checks whether referenced images exist, then compares the desired volumes, instances, and ingresses against existing resources. Owned instances and ingresses that are no longer declared in the file are planned for deletion (pruning); resources without compose ownership tags are never touched.

`up` applies the plan in order:

1. build Dockerfile services whose generated images are missing
2. ensure referenced images exist and are ready
3. create declared volumes
4. delete owned instances and ingresses that are no longer declared (pruning frees unique keys such as ingress hostnames before they are reused)
5. create or replace instances
6. create or replace ingresses

`down` deletes only instances and ingresses tagged as owned by the compose file. Volumes owned by the file are retained and reported as skipped unless `--volumes` is passed. Images are left in place because they can be shared by normal `hypeman run` usage or other compose files.

Instances and ingresses get compose ownership tags:

```text
hypeman.compose.name
hypeman.compose.service
hypeman.compose.resource
hypeman.compose.hash
```

Volumes get the same ownership tags except `hypeman.compose.service`, because a volume can be shared by multiple services.

The hash is computed from the rendered resource spec before ownership tags are added. Re-running the same file is idempotent: matching resources are reported as unchanged, changed managed resources require `--replace`, and unmanaged resources with the same name are reported as conflicts.

### Retained Volumes

Top-level `volumes` declare named volumes backed by the Hypeman volume API. Services attach them with `volumes` mount declarations:

```yaml
version: 1
name: stateful

volumes:
  data:
    size_gb: 10
  logs:
    name: stateful-logs-explicit # optional explicit name
    size_gb: 1

services:
  db:
    image: postgres:16
    volumes:
      - data:/var/lib/postgresql/data        # shorthand: volume:/abs/path[:ro|rw]
      - volume: logs                          # mapping form
        mount_path: /var/log/db
        readonly: true
```

By default, volumes are named `<compose name>-<volume key>`; set `name` for a stable external name, just like services and ingresses.

Volumes are created before the instances that mount them and are **retained**: instance replacement (via `--replace`), `compose down`, and pruning never delete them, and `compose up` on an existing volume reports it as unchanged. Deleting a retained volume requires the explicit destructive option `hypeman compose down --volumes`.

Volumes are immutable once created. Changing a declared volume (for example `size_gb`) makes `plan` report a conflict and blocks `up`, rather than silently recreating the volume and losing data. To resize, restore the original spec or delete the volume explicitly with `compose down --volumes`.

If instance replacement fails after the old instance was deleted, the retained volume and its data are untouched; re-running `compose up --replace` recreates the instance on the same volume.

Mount declarations are validated strictly: the referenced volume must be declared, the mount path must be absolute, and duplicate mount paths or mounting the same volume twice in one service are rejected.

### Strict Parsing

Compose files are parsed strictly: unknown fields and duplicate keys fail validation before any resource is applied. This applies at every level, including volume mount mappings.

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
