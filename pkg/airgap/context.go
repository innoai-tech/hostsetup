package airgap

import (
	"dagger.io/dagger"

	"github.com/innoai-tech/hostsetup/pkg/host"
)

type PrepareContext interface {
	OS() host.OS
}

type Context interface {
	OS() host.OS
	Var(k string) string
	Client() *dagger.Client
}
