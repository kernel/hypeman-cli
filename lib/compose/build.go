package compose

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kernel/hypeman-go"
	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

type desiredBuild struct {
	Service           string
	Image             string
	Hash              string
	ImageRef          string
	DockerfilePath    string
	DockerfileContent string
	Source            []byte
}

func (r *Runner) desiredBuildForService(serviceName string, service composeServiceSpec) (desiredBuild, error) {
	dockerfilePath := service.Dockerfile
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(filepath.Dir(r.file), dockerfilePath)
	}
	dockerfilePath, err := filepath.Abs(dockerfilePath)
	if err != nil {
		return desiredBuild{}, fmt.Errorf("service %q dockerfile: %w", serviceName, err)
	}
	dockerfileContent, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return desiredBuild{}, fmt.Errorf("service %q dockerfile: %w", serviceName, err)
	}
	contextPath := filepath.Dir(dockerfilePath)
	dockerignoreContent, err := readOptionalFile(filepath.Join(contextPath, ".dockerignore"))
	if err != nil {
		return desiredBuild{}, fmt.Errorf("service %q .dockerignore: %w", serviceName, err)
	}
	source, err := createSourceTarball(contextPath, dockerignoreContent)
	if err != nil {
		return desiredBuild{}, fmt.Errorf("service %q build context: %w", serviceName, err)
	}
	hash := buildHash(source, dockerfileContent, dockerignoreContent)
	image := composeBuildImageName(r.spec.Name, serviceName, hash)
	return desiredBuild{
		Service:           serviceName,
		Image:             image,
		Hash:              hash,
		DockerfilePath:    dockerfilePath,
		DockerfileContent: string(dockerfileContent),
		Source:            source,
	}, nil
}

func (r *Runner) planBuild(ctx context.Context, build desiredBuild) (Action, error) {
	action := Action{
		Type:       "build",
		Name:       build.Image,
		Service:    build.Service,
		buildInput: &build,
	}

	readyBuild, err := r.findReadyBuild(ctx, build)
	if err != nil {
		return Action{}, err
	}
	if readyBuild != nil {
		action.Action = "unchanged"
		action.Reason = "build already ready"
		action.buildInput.ImageRef = runnableBuildImage(readyBuild)
		return action, nil
	}

	action.Action = "create"
	action.Reason = "build missing"
	return action, nil
}

func (r *Runner) findReadyBuild(ctx context.Context, build desiredBuild) (*hypeman.Build, error) {
	builds, err := r.client.Builds.List(ctx, hypeman.BuildListParams{
		Tags: composeTags(r.spec.Name, build.Service, composeResourceBuild, build.Hash),
	}, r.opts...)
	if err != nil {
		return nil, fmt.Errorf("list builds for %s: %w", build.Image, err)
	}
	if builds == nil {
		return nil, nil
	}
	var ready []hypeman.Build
	for _, existing := range *builds {
		if existing.Status == hypeman.BuildStatusReady && runnableBuildImage(&existing) != "" {
			ready = append(ready, existing)
		}
	}
	if len(ready) == 0 {
		return nil, nil
	}
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].CreatedAt.After(ready[j].CreatedAt)
	})
	return &ready[0], nil
}

func (r *Runner) runBuild(ctx context.Context, build desiredBuild, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[build] image %s from %s\n", build.Image, build.DockerfilePath)
	}
	tags, err := json.Marshal(composeTags(r.spec.Name, build.Service, composeResourceBuild, build.Hash))
	if err != nil {
		return "", err
	}
	started, err := r.client.Builds.New(ctx, hypeman.BuildNewParams{
		Source:     bytes.NewReader(build.Source),
		Dockerfile: hypeman.Opt(build.DockerfileContent),
		Tags:       hypeman.Opt(string(tags)),
	}, r.opts...)
	if err != nil {
		return "", fmt.Errorf("start build %s: %w", build.Image, err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[wait] build %s ready\n", started.ID)
	}
	readyBuild, err := r.waitBuildReady(ctx, started.ID)
	if err != nil {
		return "", err
	}
	imageRef := runnableBuildImage(readyBuild)
	if imageRef == "" {
		return "", fmt.Errorf("build %s did not report a runnable image", started.ID)
	}
	return imageRef, nil
}

func (r *Runner) waitBuildReady(ctx context.Context, buildID string) (*hypeman.Build, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		build, err := r.client.Builds.Get(ctx, buildID, r.opts...)
		if err != nil {
			return nil, fmt.Errorf("check build %s: %w", buildID, err)
		}
		switch build.Status {
		case hypeman.BuildStatusReady:
			return build, nil
		case hypeman.BuildStatusFailed:
			if build.Error != "" {
				return nil, fmt.Errorf("build %s failed: %s", buildID, build.Error)
			}
			return nil, fmt.Errorf("build %s failed", buildID)
		case hypeman.BuildStatusCancelled:
			return nil, fmt.Errorf("build %s was cancelled", buildID)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func runnableBuildImage(build *hypeman.Build) string {
	if build.ImageRef != "" {
		return build.ImageRef
	}
	if build.ID != "" {
		return fmt.Sprintf("docker.io/builds/%s:latest", build.ID)
	}
	return ""
}

func composeBuildImageName(composeName, serviceName, hash string) string {
	return fmt.Sprintf("compose/%s/%s:%s", composeName, serviceName, hash)
}

func createSourceTarball(contextPath string, dockerignoreContent []byte) ([]byte, error) {
	matcher, err := newDockerignoreMatcher(dockerignoreContent)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	err = filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(contextPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relSlash := filepath.ToSlash(relPath)
		base := filepath.Base(path)
		if base == ".git" || base == "node_modules" || base == "__pycache__" ||
			base == ".venv" || base == "venv" || base == "target" ||
			base == ".docker" || base == ".dockerignore" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if matcher != nil {
			ignored, err := matcher.MatchesOrParentMatches(relSlash)
			if err != nil {
				return err
			}
			if ignored {
				return nil
			}
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relSlash
		header.ModTime = time.Unix(0, 0)
		header.AccessTime = time.Unix(0, 0)
		header.ChangeTime = time.Unix(0, 0)
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = linkTarget
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tarWriter, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func newDockerignoreMatcher(content []byte) (*patternmatcher.PatternMatcher, error) {
	if len(content) == 0 {
		return nil, nil
	}
	patterns, err := ignorefile.ReadAll(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return nil, nil
	}
	return patternmatcher.New(patterns)
}

func readOptionalFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func buildHash(parts ...[]byte) string {
	sum := sha256.New()
	for _, part := range parts {
		sum.Write(part)
	}
	return hex.EncodeToString(sum.Sum(nil))[:12]
}
