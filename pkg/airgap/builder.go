package airgap

import (
	"iter"

	"github.com/innoai-tech/hostsetup/pkg/container"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type Builder interface {
	OS() host.OS
	WorkDir() string

	Setup(ctx Context) iter.Seq[container.Step]
	AddSource(ctx Context, source host.Source) iter.Seq[container.Step]
	UpdateRepo(ctx Context, deps iter.Seq[host.Package]) iter.Seq[container.Step]

	DownloadPackages(ctx Context, deps iter.Seq[host.Package]) iter.Seq[container.Step]
	BuildOfflineRepo(ctx Context, actions iter.Seq[Action]) iter.Seq[container.Step]
}

type CanSupported interface {
	Supported(ctx PrepareContext) bool
}

type WithImageBuilder interface {
	BuildImage(ctx PrepareContext) string
}

type WithVarsBuilder interface {
	BuildVars(ctx PrepareContext) map[string]string
}

type WithSources interface {
	Sources(ctx Context) iter.Seq[host.Source]
}

type WithActions interface {
	Actions(ctx Context) iter.Seq[Action]
}
