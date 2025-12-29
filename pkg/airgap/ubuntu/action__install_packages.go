package ubuntu

import (
	"iter"
	"strings"

	"github.com/innoai-tech/hostsetup/pkg/airgap"
	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

func InstallPackages(packages []host.Package) airgap.Action {
	return &packagesInstaller{
		packages: packages,
	}
}

type packagesInstaller struct {
	packages []host.Package
}

func (pi *packagesInstaller) Deps(ctx airgap.Context) iter.Seq[host.Package] {
	return func(yield func(host.Package) bool) {
		for _, pkg := range pi.packages {
			if !yield(pkg) {
				return
			}
		}
	}
}

var _ airgap.WithDeps = &packagesInstaller{}

func (pi *packagesInstaller) Run() bash.Frag {
	if len(pi.packages) > 0 {
		return bash.StepSeq(
			func(yield func(frag bash.Frag) bool) {
				if !yield(bash.Runf("apt-get install -y %s", bash.FieldSeq(func(yield func(string) bool) {
					for _, pkg := range pi.packages {
						if strings.HasSuffix(pkg.Name, ".run") {
							continue
						}

						if pkg.DownloadOnly {
							continue
						}

						if !yield(pkg.Name) {
							return
						}
					}
				}))) {
					return
				}

				for _, pkg := range pi.packages {
					if strings.HasSuffix(pkg.Name, ".run") {
						if !yield(bash.Runf("chmod +x %[1]s && ./%[1]s %s", pkg.Name, bash.Fields(pkg.Args...))) {
							return
						}
					}
				}
			})
	}

	return bash.Fields()
}
