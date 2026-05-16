package compose

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kernel/hypeman-go"
)

type desiredBuild struct {
	Service           string
	Image             string
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
	source, err := createSourceTarball(filepath.Dir(dockerfilePath))
	if err != nil {
		return desiredBuild{}, fmt.Errorf("service %q build context: %w", serviceName, err)
	}
	hash := buildHash(source, dockerfileContent)
	image := composeBuildImageName(r.spec.Name, serviceName, hash)
	return desiredBuild{
		Service:           serviceName,
		Image:             image,
		DockerfilePath:    dockerfilePath,
		DockerfileContent: string(dockerfileContent),
		Source:            source,
	}, nil
}

func (r *Runner) planBuild(ctx context.Context, build desiredBuild) (Action, error) {
	_, err := r.client.Images.Get(ctx, url.PathEscape(build.Image), r.opts...)
	action := Action{
		Type:       "build",
		Name:       build.Image,
		Service:    build.Service,
		buildInput: &build,
	}
	if err == nil {
		action.Action = "unchanged"
		action.Reason = "image already exists"
		return action, nil
	}
	if isHTTPNotFound(err) {
		action.Action = "create"
		action.Reason = "image missing"
		return action, nil
	}
	return Action{}, fmt.Errorf("check build image %s: %w", build.Image, err)
}

func (r *Runner) runBuild(ctx context.Context, build desiredBuild, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "[build] image %s from %s\n", build.Image, build.DockerfilePath)
	}
	started, err := r.client.Builds.New(ctx, hypeman.BuildNewParams{
		Source:     bytes.NewReader(build.Source),
		Dockerfile: hypeman.Opt(build.DockerfileContent),
		ImageName:  hypeman.Opt(build.Image),
	}, r.opts...)
	if err != nil {
		return fmt.Errorf("start build %s: %w", build.Image, err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[wait] build %s ready\n", started.ID)
	}
	if _, err := r.waitBuildReady(ctx, started.ID); err != nil {
		return err
	}
	return r.waitBuiltImageReady(ctx, build.Image)
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

func (r *Runner) waitBuiltImageReady(ctx context.Context, image string) error {
	img, err := r.client.Images.Get(ctx, url.PathEscape(image), r.opts...)
	if err != nil {
		return fmt.Errorf("built image %s unavailable: %w", image, err)
	}
	return waitForImageReady(ctx, &r.client, img)
}

func composeBuildImageName(composeName, serviceName, hash string) string {
	return fmt.Sprintf("compose/%s/%s:%s", composeName, serviceName, hash)
}

func createSourceTarball(contextPath string) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
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
		base := filepath.Base(path)
		if base == ".git" || base == "node_modules" || base == "__pycache__" ||
			base == ".venv" || base == "venv" || base == "target" ||
			base == ".docker" || base == ".dockerignore" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)
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

func buildHash(source []byte, dockerfile []byte) string {
	sum := sha256.New()
	sum.Write(source)
	sum.Write(dockerfile)
	return hex.EncodeToString(sum.Sum(nil))[:12]
}
