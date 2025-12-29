package build

import (
	"context"
	"fmt"
	"os"
	"path"
	"slices"

	"dagger.io/dagger"

	"github.com/innoai-tech/hostsetup/pkg/airgap"
	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/container"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type builderContext struct {
	client *dagger.Client
	os     host.OS
	vars   map[string]string
	image  string
}

var _ airgap.Context = &builderContext{}

func (b *builderContext) Client() *dagger.Client {
	return b.client
}

func (b *builderContext) OS() host.OS {
	return b.os
}

func (b *builderContext) Var(k string) string {
	if b.vars == nil {
		return ""
	}
	return b.vars[k]
}

func Build(ctx context.Context, b airgap.Builder, outDir string) error {
	c := &builderContext{
		os:   b.OS(),
		vars: map[string]string{},
	}

	if canSupported, ok := b.(airgap.CanSupported); ok {
		if !canSupported.Supported(c) {
			return fmt.Errorf("%s is not supported", c.os)
		}
	}

	if fromImage, ok := b.(airgap.WithImageBuilder); ok {
		c.image = fromImage.BuildImage(c)
	} else {
		c.image = fmt.Sprintf("%s:%s", c.os.Name, c.os.Version)
	}

	if builder, ok := b.(airgap.WithVarsBuilder); ok {
		c.vars = builder.BuildVars(c)
	}

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	c.client = client

	cc := client.Container(
		dagger.ContainerOpts{
			Platform: dagger.Platform(fmt.Sprintf("linux/%s", c.os.Arch)),
		}).
		From(c.image)

	cc = container.ApplySeq(cc, b.Setup(c))

	if s, ok := b.(airgap.WithSources); ok {
		for src := range s.Sources(c) {
			cc = container.ApplySeq(cc, b.AddSource(c, src))
		}
	}

	actions := make([]airgap.Action, 0)
	depGroups := make([][]host.Package, 0)

	if s, ok := b.(airgap.WithActions); ok {
		for action := range s.Actions(c) {
			if x, ok := action.(airgap.WithDeps); ok {
				deps := slices.Collect(x.Deps(c))
				if len(deps) > 0 {
					depGroups = append(depGroups, deps)
				}
			}

			actions = append(actions, action)
		}
	}

	cc = container.ApplySeq(cc, b.UpdateRepo(c, func(yield func(host.Package) bool) {
		for _, deps := range depGroups {
			for _, dep := range deps {
				if !yield(dep) {
					return
				}
			}
		}
	}))

	for _, deps := range depGroups {
		cc = container.ApplySeq(cc, b.DownloadPackages(c, slices.Values(deps)))
	}

	if _, err := cc.Sync(ctx); err != nil {
		return err
	}

	cc = container.ApplySeq(cc, b.BuildOfflineRepo(c, func(yield func(airgap.Action) bool) {
		for _, a := range actions {
			if !yield(a) {
				return
			}
		}

		if !yield(airgap.Steps(
			bash.Run(`
AVAILABLE_FUNCS=$(declare -F | awk '{print $3}' | grep -v "^_")

function usage() {
	echo "用法: $0 [命令]"
	echo "------------------------------------------------"
	echo "可用命令列表:"
	echo "${AVAILABLE_FUNCS}" | sed 's/^/  - /'
	echo "------------------------------------------------"
	echo "示例: bash $0 init"
}

if [ -z "$1" ] || ! echo "${AVAILABLE_FUNCS}" | grep -qw "$1"; then
	usage
	exit 1
fi

"$@"
`),
		)) {
		}
	}))

	if _, err := cc.
		Directory(b.WorkDir()).
		Export(
			ctx,
			path.Join(outDir, fmt.Sprintf("%s-%s", c.os.DistroRelease(), c.os.Arch)),
			dagger.DirectoryExportOpts{
				Wipe: true,
			},
		); err != nil {
		return err
	}

	return nil
}
