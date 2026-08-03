package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposeVolumeMountShorthand(t *testing.T) {
	mount, err := parseComposeVolumeMountShorthand("data:/var/lib/data")
	require.NoError(t, err)
	assert.Equal(t, composeVolumeMountSpec{Volume: "data", MountPath: "/var/lib/data"}, mount)

	mount, err = parseComposeVolumeMountShorthand("data:/var/lib/data:ro")
	require.NoError(t, err)
	assert.Equal(t, composeVolumeMountSpec{Volume: "data", MountPath: "/var/lib/data", Readonly: true}, mount)

	mount, err = parseComposeVolumeMountShorthand("data:/var/lib/data:rw")
	require.NoError(t, err)
	assert.Equal(t, composeVolumeMountSpec{Volume: "data", MountPath: "/var/lib/data"}, mount)

	for _, invalid := range []string{
		"data",
		"data:",
		":/var/lib/data",
		"data:/var/lib/data:ro:extra",
		"data:/var/lib/data:readonly",
		"data:/var/lib/data:",
	} {
		_, err := parseComposeVolumeMountShorthand(invalid)
		assert.Error(t, err, "expected %q to be rejected", invalid)
	}
}

func TestLoadComposeSpecRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: worker-stack
surprise: true
services:
  worker:
    image: alpine:latest
`), 0644))

	_, err := loadComposeSpec(composePath)
	require.ErrorContains(t, err, "field surprise not found")

	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: worker-stack
services:
  worker:
    image: alpine:latest
    bogus: 1
`), 0644))
	_, err = loadComposeSpec(composePath)
	require.ErrorContains(t, err, "field bogus not found")

	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: worker-stack
volumes:
  data:
    size_gb: 5
services:
  worker:
    image: alpine:latest
    volumes:
      - volume: data
        mount_path: /var/lib/data
        encrypt: true
`), 0644))
	_, err = loadComposeSpec(composePath)
	require.ErrorContains(t, err, "field encrypt not found")
}

func TestLoadComposeSpecRejectsDuplicateKeys(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: worker-stack
name: other-stack
services:
  worker:
    image: alpine:latest
`), 0644))

	_, err := loadComposeSpec(composePath)
	require.ErrorContains(t, err, "already defined")
}

func TestLoadComposeSpecAcceptsVolumes(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: stateful
volumes:
  data:
    size_gb: 5
  logs:
    name: stateful-logs-explicit
    size_gb: 1
services:
  db:
    image: postgres:16
    volumes:
      - data:/var/lib/postgresql/data
      - volume: logs
        mount_path: /var/log/db
        readonly: true
`), 0644))

	spec, err := loadComposeSpec(composePath)
	require.NoError(t, err)

	require.Len(t, spec.Volumes, 2)
	assert.Equal(t, int64(5), spec.Volumes["data"].SizeGB)
	assert.Equal(t, "stateful-logs-explicit", spec.Volumes["logs"].Name)

	service := spec.Services["db"]
	require.Len(t, service.Volumes, 2)
	assert.Equal(t, composeVolumeMountSpec{Volume: "data", MountPath: "/var/lib/postgresql/data"}, service.Volumes[0])
	assert.Equal(t, composeVolumeMountSpec{Volume: "logs", MountPath: "/var/log/db", Readonly: true}, service.Volumes[1])
}

func TestValidateComposeSpecVolumes(t *testing.T) {
	base := func() *composeSpec {
		return &composeSpec{
			Version: 1,
			Name:    "stateful",
			Volumes: map[string]composeVolumeSpec{
				"data": {SizeGB: 5},
			},
			Services: map[string]composeServiceSpec{
				"db": {
					Image:   "postgres:16",
					Volumes: []composeVolumeMountSpec{{Volume: "data", MountPath: "/var/lib/postgresql/data"}},
				},
			},
		}
	}

	require.NoError(t, validateComposeSpec(base()))

	cases := []struct {
		name    string
		mutate  func(*composeSpec)
		wantErr string
	}{
		{
			name: "invalid volume key",
			mutate: func(s *composeSpec) {
				s.Volumes = map[string]composeVolumeSpec{"BadName": {SizeGB: 5}}
			},
			wantErr: `volume "BadName" must contain only lowercase letters, digits, and dashes`,
		},
		{
			name: "invalid explicit volume name",
			mutate: func(s *composeSpec) {
				s.Volumes = map[string]composeVolumeSpec{"data": {Name: "BadName", SizeGB: 5}}
			},
			wantErr: `volume "data" name must contain only lowercase letters, digits, and dashes`,
		},
		{
			name: "duplicate resolved volume names",
			mutate: func(s *composeSpec) {
				s.Volumes = map[string]composeVolumeSpec{
					"data":  {SizeGB: 5},
					"other": {Name: "stateful-data", SizeGB: 5},
				}
			},
			wantErr: `produces duplicate volume name "stateful-data"`,
		},
		{
			name: "missing size",
			mutate: func(s *composeSpec) {
				s.Volumes = map[string]composeVolumeSpec{"data": {}}
			},
			wantErr: `volume "data" size_gb must be positive`,
		},
		{
			name: "mount references unknown volume",
			mutate: func(s *composeSpec) {
				service := s.Services["db"]
				service.Volumes = []composeVolumeMountSpec{{Volume: "missing", MountPath: "/data"}}
				s.Services["db"] = service
			},
			wantErr: `service "db" volume mount 0 references unknown volume "missing"`,
		},
		{
			name: "mount missing volume",
			mutate: func(s *composeSpec) {
				service := s.Services["db"]
				service.Volumes = []composeVolumeMountSpec{{MountPath: "/data"}}
				s.Services["db"] = service
			},
			wantErr: `service "db" volume mount 0 volume is required`,
		},
		{
			name: "mount missing path",
			mutate: func(s *composeSpec) {
				service := s.Services["db"]
				service.Volumes = []composeVolumeMountSpec{{Volume: "data"}}
				s.Services["db"] = service
			},
			wantErr: `service "db" volume mount 0 mount_path is required`,
		},
		{
			name: "mount path must be absolute",
			mutate: func(s *composeSpec) {
				service := s.Services["db"]
				service.Volumes = []composeVolumeMountSpec{{Volume: "data", MountPath: "relative/path"}}
				s.Services["db"] = service
			},
			wantErr: `mount_path "relative/path" must be an absolute path`,
		},
		{
			name: "duplicate mount path",
			mutate: func(s *composeSpec) {
				s.Volumes["logs"] = composeVolumeSpec{SizeGB: 1}
				service := s.Services["db"]
				service.Volumes = append(service.Volumes, composeVolumeMountSpec{Volume: "logs", MountPath: "/var/lib/postgresql/data"})
				s.Services["db"] = service
			},
			wantErr: `duplicates mount_path "/var/lib/postgresql/data"`,
		},
		{
			name: "same volume mounted twice",
			mutate: func(s *composeSpec) {
				service := s.Services["db"]
				service.Volumes = append(service.Volumes, composeVolumeMountSpec{Volume: "data", MountPath: "/elsewhere"})
				s.Services["db"] = service
			},
			wantErr: `mounts volume "data" more than once`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := base()
			tc.mutate(spec)
			require.ErrorContains(t, validateComposeSpec(spec), tc.wantErr)
		})
	}
}

func TestDesiredResourcesRendersVolumesAndMounts(t *testing.T) {
	runner := Runner{
		spec: composeSpec{
			Version: 1,
			Name:    "stateful",
			Volumes: map[string]composeVolumeSpec{
				"data": {SizeGB: 5},
				"logs": {Name: "stateful-logs-explicit", SizeGB: 1},
			},
			Services: map[string]composeServiceSpec{
				"db": {
					Image: "postgres:16",
					Volumes: []composeVolumeMountSpec{
						{Volume: "data", MountPath: "/var/lib/postgresql/data"},
						{Volume: "logs", MountPath: "/var/log/db", Readonly: true},
					},
				},
			},
		},
	}

	_, volumes, instances, _, _, err := runner.desiredResources()
	require.NoError(t, err)

	require.Len(t, volumes, 2)
	assert.Equal(t, "stateful-data", volumes[0].Name)
	assert.Equal(t, int64(5), volumes[0].Input.SizeGB)
	assert.Equal(t, composeResourceVolume, volumes[0].Input.Tags[composeTagResource])
	assert.NotEmpty(t, volumes[0].Input.Tags[composeTagHash])
	assert.Equal(t, "stateful-logs-explicit", volumes[1].Name)

	require.Len(t, instances, 1)
	require.Len(t, instances[0].Input.Volumes, 2)
	// Mounts carry the volume name (not the server ID) so the rendered hash is
	// stable before the volume exists.
	assert.Equal(t, "stateful-data", instances[0].Input.Volumes[0].VolumeID)
	assert.Equal(t, "/var/lib/postgresql/data", instances[0].Input.Volumes[0].MountPath)
	assert.Equal(t, "stateful-logs-explicit", instances[0].Input.Volumes[1].VolumeID)
	assert.True(t, instances[0].Input.Volumes[1].Readonly.Valid())

	// Rendering is deterministic: identical input yields identical hashes.
	_, volumesAgain, instancesAgain, _, _, err := runner.desiredResources()
	require.NoError(t, err)
	require.Equal(t, volumes[0].Hash, volumesAgain[0].Hash)
	require.Equal(t, instances[0].Hash, instancesAgain[0].Hash)
}

func TestPlanVolumeAction(t *testing.T) {
	desired := desiredVolume{
		Name: "stateful-data",
		Hash: "hash-a",
		Input: hypeman.VolumeNewParams{
			Name:   "stateful-data",
			SizeGB: 5,
		},
	}

	action := planVolumeAction(desired, nil, nil)
	assert.Equal(t, "create", action.Action)
	assert.Equal(t, "missing", action.Reason)

	owned := []hypeman.Volume{{
		ID:   "vol-1",
		Name: "stateful-data",
		Tags: composeVolumeTags("stateful", "hash-a"),
	}}
	action = planVolumeAction(desired, owned, nil)
	assert.Equal(t, "unchanged", action.Action)
	assert.Equal(t, "vol-1", action.volumeID)

	// A spec change is a conflict, never a replace: replacing would destroy data.
	owned[0].Tags = composeVolumeTags("stateful", "hash-b")
	action = planVolumeAction(desired, owned, nil)
	assert.Equal(t, "conflict", action.Action)
	assert.Contains(t, action.Reason, "immutable")
	assert.Equal(t, "vol-1", action.volumeID)

	// An unmanaged volume with the same name is a conflict.
	action = planVolumeAction(desired, nil, []hypeman.Volume{{ID: "vol-9", Name: "stateful-data"}})
	assert.Equal(t, "conflict", action.Action)
	assert.Equal(t, "name exists without compose ownership", action.Reason)
}

// fakeHypeman is an in-memory Hypeman API seam covering the endpoints compose
// uses. Volumes carry a data payload so tests can prove data (a nonce written
// by the guest) survives instance replacement and down/up.
type fakeHypeman struct {
	t *testing.T

	mu        sync.Mutex
	volumes   map[string]*fakeVolume
	instances map[string]*fakeInstance
	nextID    int
	requests  []string

	failInstanceCreates int
}

type fakeVolume struct {
	id     string
	name   string
	sizeGB int64
	tags   map[string]string
	data   string
}

type fakeInstance struct {
	id      string
	name    string
	image   string
	tags    map[string]string
	volumes []hypeman.VolumeMountParam
}

func newFakeHypeman(t *testing.T) (*fakeHypeman, *httptest.Server) {
	t.Helper()
	fake := &fakeHypeman{
		t:         t,
		volumes:   map[string]*fakeVolume{},
		instances: map[string]*fakeInstance{},
	}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	return fake, server
}

func (f *fakeHypeman) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%d", prefix, f.nextID)
}

func (f *fakeHypeman) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/")
	switch {
	case path == "volumes" && r.Method == http.MethodGet:
		f.listVolumes(w, r)
	case path == "volumes" && r.Method == http.MethodPost:
		f.createVolume(w, r)
	case strings.HasPrefix(path, "volumes/") && r.Method == http.MethodDelete:
		f.deleteVolume(w, strings.TrimPrefix(path, "volumes/"))
	case path == "instances" && r.Method == http.MethodGet:
		f.listInstances(w, r)
	case path == "instances" && r.Method == http.MethodPost:
		f.createInstance(w, r)
	case strings.HasPrefix(path, "instances/") && r.Method == http.MethodDelete:
		f.deleteInstance(w, strings.TrimPrefix(path, "instances/"))
	case path == "ingresses" && r.Method == http.MethodGet:
		writeJSON(w, []any{})
	case strings.HasPrefix(path, "images/") && r.Method == http.MethodGet:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	case path == "images" && r.Method == http.MethodPost:
		var body map[string]any
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, map[string]any{
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"digest":     "sha256:fake",
			"name":       body["name"],
			"status":     "ready",
		})
	default:
		http.Error(w, fmt.Sprintf(`{"error":"unhandled %s %s"}`, r.Method, r.URL.Path), http.StatusNotFound)
	}
}

func tagFilter(r *http.Request) map[string]string {
	tags := map[string]string{}
	for key, values := range r.URL.Query() {
		if !strings.HasPrefix(key, "tags[") || !strings.HasSuffix(key, "]") || len(values) == 0 {
			continue
		}
		tags[strings.TrimSuffix(strings.TrimPrefix(key, "tags["), "]")] = values[0]
	}
	return tags
}

func tagsMatch(resource, filter map[string]string) bool {
	for key, value := range filter {
		if resource[key] != value {
			return false
		}
	}
	return true
}

func (f *fakeHypeman) listVolumes(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	filter := tagFilter(r)
	out := []map[string]any{}
	for _, vol := range f.volumes {
		if !tagsMatch(vol.tags, filter) {
			continue
		}
		out = append(out, map[string]any{
			"id":         vol.id,
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"name":       vol.name,
			"size_gb":    vol.sizeGB,
			"tags":       vol.tags,
		})
	}
	writeJSON(w, out)
}

func (f *fakeHypeman) createVolume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string            `json:"name"`
		SizeGB int64             `json:"size_gb"`
		Tags   map[string]string `json:"tags"`
	}
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
	f.mu.Lock()
	defer f.mu.Unlock()
	vol := &fakeVolume{id: f.id("vol"), name: body.Name, sizeGB: body.SizeGB, tags: body.Tags}
	f.volumes[vol.id] = vol
	writeJSON(w, map[string]any{
		"id":         vol.id,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"name":       vol.name,
		"size_gb":    vol.sizeGB,
		"tags":       vol.tags,
	})
}

func (f *fakeHypeman) deleteVolume(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.volumes[id]; !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.volumes, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeHypeman) listInstances(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	filter := tagFilter(r)
	out := []map[string]any{}
	for _, inst := range f.instances {
		if !tagsMatch(inst.tags, filter) {
			continue
		}
		out = append(out, map[string]any{
			"id":         inst.id,
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"image":      inst.image,
			"name":       inst.name,
			"state":      "Running",
			"tags":       inst.tags,
		})
	}
	writeJSON(w, out)
}

func (f *fakeHypeman) createInstance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string                     `json:"name"`
		Image   string                     `json:"image"`
		Tags    map[string]string          `json:"tags"`
		Volumes []hypeman.VolumeMountParam `json:"volumes"`
	}
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failInstanceCreates > 0 {
		f.failInstanceCreates--
		http.Error(w, `{"error":"simulated instance create failure"}`, http.StatusInternalServerError)
		return
	}
	inst := &fakeInstance{id: f.id("inst"), name: body.Name, image: body.Image, tags: body.Tags, volumes: body.Volumes}
	f.instances[inst.id] = inst
	writeJSON(w, map[string]any{
		"id":         inst.id,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"image":      inst.image,
		"name":       inst.name,
		"state":      "Running",
		"tags":       inst.tags,
	})
}

func (f *fakeHypeman) deleteInstance(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.instances[id]; !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.instances, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeHypeman) onlyInstance() *fakeInstance {
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Len(f.t, f.instances, 1)
	for _, inst := range f.instances {
		return inst
	}
	return nil
}

func (f *fakeHypeman) volumeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.volumes)
}

func (f *fakeHypeman) onlyVolume() *fakeVolume {
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Len(f.t, f.volumes, 1)
	for _, vol := range f.volumes {
		return vol
	}
	return nil
}

func (f *fakeHypeman) countRequests(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, req := range f.requests {
		if strings.HasPrefix(req, prefix) {
			count++
		}
	}
	return count
}

func (f *fakeHypeman) requestIndex(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, req := range f.requests {
		if strings.HasPrefix(req, prefix) {
			return i
		}
	}
	return -1
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeComposeFile(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0644))
	return path
}

func newTestRunner(t *testing.T, composePath string, server *httptest.Server) *Runner {
	t.Helper()
	client := hypeman.NewClient(option.WithBaseURL(server.URL))
	runner, err := NewRunner(composePath, client)
	require.NoError(t, err)
	return runner
}

const statefulComposeFile = `
version: 1
name: stateful
volumes:
  data:
    size_gb: 5
services:
  db:
    image: postgres:16
    volumes:
      - data:/var/lib/postgresql/data
`

func TestComposeUpDownUpRetainsVolumeAndNonce(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, statefulComposeFile)
	ctx := context.Background()

	runner := newTestRunner(t, composePath, server)
	plan, err := runner.Up(ctx, UpOptions{})
	require.NoError(t, err)
	assert.Equal(t, 3, plan.Summary.Create) // image + volume + instance

	// Volumes are created before instances.
	volumeCreate := fake.requestIndex("POST /volumes")
	instanceCreate := fake.requestIndex("POST /instances")
	require.GreaterOrEqual(t, volumeCreate, 0)
	require.GreaterOrEqual(t, instanceCreate, 0)
	assert.Less(t, volumeCreate, instanceCreate)

	vol := fake.onlyVolume()
	assert.Equal(t, "stateful-data", vol.name)
	inst := fake.onlyInstance()
	require.Len(t, inst.volumes, 1)
	assert.Equal(t, vol.id, inst.volumes[0].VolumeID)
	assert.Equal(t, "/var/lib/postgresql/data", inst.volumes[0].MountPath)

	// The guest writes a nonce to its volume.
	vol.data = "nonce-abc123"

	// Down retains the volume by default and says so in the plan.
	downPlan, err := newTestRunner(t, composePath, server).Down(ctx, DownOptions{})
	require.NoError(t, err)
	assert.Empty(t, fake.instances)
	require.Equal(t, 1, fake.volumeCount())
	var retainAction *Action
	for i := range downPlan.Actions {
		if downPlan.Actions[i].Type == "volume" {
			retainAction = &downPlan.Actions[i]
		}
	}
	require.NotNil(t, retainAction)
	assert.Equal(t, "skip", retainAction.Action)
	assert.Contains(t, retainAction.Reason, "retained")

	// Up again reuses the retained volume: no new volume is created and the
	// new instance mounts the same volume ID, so the nonce survives down/up.
	volumeCreatesBefore := fake.countRequests("POST /volumes")
	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	assert.Equal(t, volumeCreatesBefore, fake.countRequests("POST /volumes"))
	inst = fake.onlyInstance()
	require.Len(t, inst.volumes, 1)
	assert.Equal(t, vol.id, inst.volumes[0].VolumeID)
	assert.Equal(t, "nonce-abc123", fake.onlyVolume().data)
}

func TestComposeReplaceRetainsVolumeAndNonce(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, statefulComposeFile)
	ctx := context.Background()

	_, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	vol := fake.onlyVolume()
	vol.data = "nonce-persist-me"
	firstInstanceID := fake.onlyInstance().id

	// Change the rendered spec so the instance must be replaced.
	writeComposeFile(t, dir, `
version: 1
name: stateful
volumes:
  data:
    size_gb: 5
services:
  db:
    image: postgres:16
    env:
      POSTGRES_PASSWORD: changed
    volumes:
      - data:/var/lib/postgresql/data
`)

	// Without --replace, up refuses and touches nothing.
	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.ErrorContains(t, err, "replace required")
	assert.Equal(t, firstInstanceID, fake.onlyInstance().id)

	plan, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{Replace: true})
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.Replace)

	// The replacement instance mounts the same retained volume; the nonce
	// survives replacement and the volume itself was never recreated.
	inst := fake.onlyInstance()
	assert.NotEqual(t, firstInstanceID, inst.id)
	require.Len(t, inst.volumes, 1)
	assert.Equal(t, vol.id, inst.volumes[0].VolumeID)
	require.Equal(t, 1, fake.volumeCount())
	assert.Equal(t, "nonce-persist-me", fake.onlyVolume().data)
	assert.Equal(t, 0, fake.countRequests("DELETE /volumes"))
}

func TestComposeReplaceFailureLeavesVolumeRecoverable(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, statefulComposeFile)
	ctx := context.Background()

	_, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	vol := fake.onlyVolume()
	vol.data = "nonce-recover-me"

	writeComposeFile(t, dir, `
version: 1
name: stateful
volumes:
  data:
    size_gb: 5
services:
  db:
    image: postgres:16
    env:
      POSTGRES_PASSWORD: changed
    volumes:
      - data:/var/lib/postgresql/data
`)

	// The old instance is deleted but creation of the replacement fails.
	// (The SDK retries 5xx, so fail enough times to exhaust retries.)
	fake.failInstanceCreates = 10
	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{Replace: true})
	require.Error(t, err)
	assert.Empty(t, fake.instances)

	// The retained volume and its data are untouched and recoverable.
	require.Equal(t, 1, fake.volumeCount())
	assert.Equal(t, "nonce-recover-me", fake.onlyVolume().data)

	// Retrying up recreates the instance on the same retained volume.
	fake.failInstanceCreates = 0
	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{Replace: true})
	require.NoError(t, err)
	inst := fake.onlyInstance()
	require.Len(t, inst.volumes, 1)
	assert.Equal(t, vol.id, inst.volumes[0].VolumeID)
	assert.Equal(t, "nonce-recover-me", fake.onlyVolume().data)
}

func TestComposeDownVolumesDeletesRetainedData(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, statefulComposeFile)
	ctx := context.Background()

	_, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	fake.onlyVolume().data = "nonce-doomed"

	plan, err := newTestRunner(t, composePath, server).Down(ctx, DownOptions{Volumes: true})
	require.NoError(t, err)
	assert.Empty(t, fake.instances)
	assert.Empty(t, fake.volumes)
	var volumeDelete *Action
	for i := range plan.Actions {
		if plan.Actions[i].Type == "volume" {
			volumeDelete = &plan.Actions[i]
		}
	}
	require.NotNil(t, volumeDelete)
	assert.Equal(t, "delete", volumeDelete.Action)
	assert.Contains(t, volumeDelete.Reason, "destroys retained data")

	// Down is idempotent: running it again finds nothing.
	plan, err = newTestRunner(t, composePath, server).Down(ctx, DownOptions{Volumes: true})
	require.NoError(t, err)
	for _, action := range plan.Actions {
		assert.Equal(t, "skip", action.Action)
		assert.Equal(t, "not found", action.Reason)
	}
}

func TestComposeVolumeSpecChangeConflictsWithoutTouchingData(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, statefulComposeFile)
	ctx := context.Background()

	_, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	fake.onlyVolume().data = "nonce-safe"

	// Changing the declared volume size must not resize/replace the volume.
	writeComposeFile(t, dir, strings.Replace(statefulComposeFile, "size_gb: 5", "size_gb: 10", 1))
	plan, err := newTestRunner(t, composePath, server).Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.Conflict)

	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{Replace: true})
	require.ErrorContains(t, err, "conflicts found")
	require.Equal(t, 1, fake.volumeCount())
	assert.Equal(t, int64(5), fake.onlyVolume().sizeGB)
	assert.Equal(t, "nonce-safe", fake.onlyVolume().data)
	assert.Equal(t, 0, fake.countRequests("DELETE /volumes"))
}

func TestComposeUpPrunesRemovedOwnedResourcesOnly(t *testing.T) {
	fake, server := newFakeHypeman(t)
	dir := t.TempDir()
	composePath := writeComposeFile(t, dir, `
version: 1
name: stateful
volumes:
  data:
    size_gb: 5
services:
  db:
    image: postgres:16
    volumes:
      - data:/var/lib/postgresql/data
  cache:
    image: redis:7
`)
	ctx := context.Background()

	_, err := newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	require.Len(t, fake.instances, 2)

	// An unmanaged instance with no compose ownership tags exists alongside.
	fake.mu.Lock()
	fake.instances["unmanaged-1"] = &fakeInstance{id: "unmanaged-1", name: "stateful-cache-lookalike", image: "redis:7"}
	fake.mu.Unlock()

	// Remove the cache service from the file.
	writeComposeFile(t, dir, statefulComposeFile)
	plan, err := newTestRunner(t, composePath, server).Plan(ctx)
	require.NoError(t, err)
	var pruned []Action
	for _, action := range plan.Actions {
		if action.Action == "delete" {
			pruned = append(pruned, action)
		}
	}
	require.Len(t, pruned, 1)
	assert.Equal(t, "instance", pruned[0].Type)
	assert.Equal(t, "stateful-cache", pruned[0].Name)
	assert.Equal(t, "no longer declared in compose file", pruned[0].Reason)

	_, err = newTestRunner(t, composePath, server).Up(ctx, UpOptions{})
	require.NoError(t, err)
	fake.mu.Lock()
	_, cacheExists := fake.instancesByName("stateful-cache")
	_, unmanagedAlive := fake.instances["unmanaged-1"]
	dbCount := len(fake.instances)
	fake.mu.Unlock()
	assert.False(t, cacheExists)
	assert.True(t, unmanagedAlive, "unmanaged instance must not be pruned")
	assert.Equal(t, 2, dbCount) // db instance + unmanaged
}

func (f *fakeHypeman) instancesByName(name string) (*fakeInstance, bool) {
	for _, inst := range f.instances {
		if inst.name == name {
			return inst, true
		}
	}
	return nil, false
}
