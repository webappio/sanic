package buildkit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/containerd/containerd/platforms"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

const (
	buildArgPrefix       = "build-arg:"
	labelPrefix          = "label:"
)

//BuilderOpts represent options that can be set to configure buildkit's processing of this dockerfile
type BuilderOpts struct {
	ContextDir     string
	Target         string
	DockerfileName string
	ResolveMode    llb.ResolveMode
	NetworkMode    pb.NetMode
	IgnoreCache    []string
	CacheImports   []client.CacheOptionsEntry
}

func setDefaults(buildopts *BuilderOpts) {
	if buildopts.DockerfileName == "" {
		buildopts.DockerfileName = "Dockerfile"
	}
	if buildopts.IgnoreCache == nil {
		buildopts.IgnoreCache = []string{}
	}
	if buildopts.CacheImports == nil {
		buildopts.CacheImports = []client.CacheOptionsEntry{}
	}
}

func loadMainFiles(
	ctx context.Context,
	buildopts *BuilderOpts,
	c client.Client,
	marshalOpts []llb.ConstraintsOpt,
) (dtDockerfile, dtDockerignore []byte, err error) {
	src := llb.Local("dockerfile",
		llb.FollowPaths([]string{buildopts.DockerfileName, ".dockerfile"}),
		llb.SessionID(c.BuildOpts().SessionID),
		llb.SharedKeyHint("dockerfile"),
		dockerfile2llb.WithInternalName("load build definition from "+buildopts.DockerfileName),
	)

	def, err := src.Marshal(marshalOpts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to marshal local source")
	}

	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to resolve dockerfile")
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, nil, err
	}

	dtDockerfile, err = ref.ReadFile(ctx, client.ReadRequest{
		Filename: buildopts.DockerfileName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read dockerfile")
	}

	dt, err := ref.ReadFile(ctx, client.ReadRequest{
		Filename: buildopts.DockerfileName + ".dockerignore",
	})
	if err == nil {
		dtDockerignore = dt
	} else {
		dt, err := ref.ReadFile(ctx, client.ReadRequest{
			Filename: ".dockerignore",
		})
		if err == nil {
			dtDockerignore = dt
		}
	}
	err = nil
	return
}

//Builder returns a buildkit compatible BuildFunc
func Builder(buildopts *BuilderOpts) func(ctx context.Context, c client.Client) (*client.Result, error) {
	setDefaults(buildopts)
	return func(ctx context.Context, c client.Client) (*client.Result, error) {
		opts := c.BuildOpts().Opts
		caps := c.BuildOpts().LLBCaps
		var err error

		marshalOpts := []llb.ConstraintsOpt{llb.WithCaps(caps)}

		defaultBuildPlatform := platforms.DefaultSpec()
		if workers := c.BuildOpts().Workers; len(workers) > 0 && len(workers[0].Platforms) > 0 {
			defaultBuildPlatform = workers[0].Platforms[0]
		}
		buildPlatforms := []specs.Platform{defaultBuildPlatform}

		dtDockerfile, dtDockerignore, err := loadMainFiles(
			ctx,
			buildopts,
			c,
			marshalOpts,
		)
		if err != nil {
			return nil, err
		}

		var excludes []string
		if dtDockerignore != nil {
			excludes, err = dockerignore.ReadAll(bytes.NewBuffer(dtDockerignore))
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse dockerignore")
			}
		}

		res := client.NewResult()

		st, img, err := dockerfile2llb.Dockerfile2LLB(ctx, dtDockerfile, dockerfile2llb.ConvertOpt{
			Target:            buildopts.Target,
			MetaResolver:      c,
			BuildArgs:         filter(opts, buildArgPrefix),
			Labels:            filter(opts, labelPrefix),
			SessionID:         c.BuildOpts().SessionID,
			Excludes:          excludes,
			IgnoreCache:       buildopts.IgnoreCache,
			BuildPlatforms:    buildPlatforms,
			ImageResolveMode:  buildopts.ResolveMode,
			ForceNetMode:      buildopts.NetworkMode,
			LLBCaps:           &caps,
		})

		if err != nil {
			return nil, errors.Wrapf(err, "failed to create LLB definition")
		}

		def, err := st.Marshal()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal LLB definition")
		}

		config, err := json.Marshal(img)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal image config")
		}

		r, err := c.Solve(ctx, client.SolveRequest{
			Definition:   def.ToPB(),
			CacheImports: buildopts.CacheImports,
		})
		if err != nil {
			return nil, err
		}

		ref, err := r.SingleRef()
		if err != nil {
			return nil, err
		}

		res.AddMeta(exptypes.ExporterImageConfigKey, config)
		res.SetRef(ref)

		return res, nil
	}
}

func filter(opt map[string]string, key string) map[string]string {
	m := map[string]string{}
	for k, v := range opt {
		if strings.HasPrefix(k, key) {
			m[strings.TrimPrefix(k, key)] = v
		}
	}
	return m
}
