package ubuntu

import (
	"cmp"
	"fmt"
	"iter"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/innoai-tech/hostsetup/pkg/airgap"
	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/container"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type Builder struct{}

func (Builder) WorkDir() string {
	return "/opt/offline-repo"
}

var _ airgap.Builder = Builder{}

func (Builder) OS() host.OS {
	return host.OS{
		Name:    "ubuntu",
		Version: cmp.Or(os.Getenv("TARGET_VERSION"), "22.04"),
		Arch:    cmp.Or(os.Getenv("TARGET_ARCH"), runtime.GOARCH),
	}
}

var _ airgap.WithImageBuilder = Builder{}

func (Builder) BuildImage(ctx airgap.PrepareContext) string {
	return fmt.Sprintf("ubuntu:%s", ctx.OS().Version)
}

func (b Builder) Setup(ctx airgap.Context) iter.Seq[container.Step] {
	return container.Of(
		container.Env{
			"DEBIAN_FRONTEND": "noninteractive",
		},
		&container.Run{
			WorkDir: b.WorkDir(),
			Script: bash.Steps(
				bash.Run("apt-get update"),
				bash.Runf("apt-get install -y --no-install-recommends %s", bash.Fields(
					"wget",
					"dpkg-dev",
					"ca-certificates",
					"software-properties-common",
					"gnupg2",
				)),
			),
		},
	)
}

func (Builder) AddSource(ctx airgap.Context, source host.Source) iter.Seq[container.Step] {
	client := ctx.Client()

	return func(yield func(container.Step) bool) {
		switch path.Ext(source.Url) {
		case ".list":
			if !yield(&container.File{
				Path:   fmt.Sprintf("/etc/apt/sources.list.d/%s.list", source.Name),
				Source: client.HTTP(source.Url),
			}) {
				return
			}
		case ".deb":
			// local repo
			debFilename := fmt.Sprintf("/opt/%s.deb", source.Name)

			if !yield(&container.File{
				Path:   debFilename,
				Source: client.HTTP(source.Url),
			}) {
				return
			}

			if !yield(&container.Run{
				Script: bash.Steps(
					bash.Runf("dpkg -i %s", debFilename),
					bash.Runf("cp /var/%s-*/*-keyring.gpg /usr/share/keyrings/", source.Name),
				),
			}) {
				return
			}
		default:

		}

		if len(source.Contents) > 0 {
			if !yield(&container.File{
				Path:     fmt.Sprintf("/etc/apt/sources.list.d/%s.list", source.Name),
				Contents: source.Contents,
			}) {
				return
			}
		}

		if source.KeyDownloadUrl != "" {
			keyFile := fmt.Sprintf("/usr/share/keyrings/%s.gpg", source.Name)

			if source.KeyType != "" {
				keyFile = fmt.Sprintf("/usr/share/keyrings/%s.%s", source.Name, source.KeyType)
			}

			if !yield(&container.File{
				Path:   keyFile,
				Source: client.HTTP(source.KeyDownloadUrl),
			}) {
				return
			}

			if !yield(&container.Run{
				Script: bash.Runf("apt-key add %s", keyFile),
			}) {
				return
			}
		}
	}
}

func (b Builder) UpdateRepo(ctx airgap.Context, deps iter.Seq[host.Package]) iter.Seq[container.Step] {
	client := ctx.Client()

	return func(yield func(container.Step) bool) {
		hasTempRepo := false

		for dep := range deps {
			if dep.DownloadURL != "" {
				if strings.HasSuffix(dep.Name, ".run") {
					if !yield(&container.File{
						Path:   dep.Name,
						Source: client.HTTP(dep.DownloadURL),
					}) {
						return
					}

					continue
				}

				if !yield(&container.File{
					Path:   fmt.Sprintf("/opt/temp-repo/debs/%s", path.Base(dep.DownloadURL)),
					Source: client.HTTP(dep.DownloadURL),
				}) {
					return
				}

				hasTempRepo = true
			}
		}

		if hasTempRepo {
			if !yield(&container.File{
				Path: "/etc/apt/sources.list.d/temp.list",
				Contents: []byte(`deb [trusted=yes] file:///opt/temp-repo ./
`),
			}) {
				return
			}

			if !yield(&container.Run{
				WorkDir: "/opt/temp-repo",
				Script:  bash.Run("dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz"),
			}) {
				return
			}
		}

		if !yield(&container.Run{
			WorkDir: b.WorkDir(),
			Script:  bash.Run("apt-get update"),
		}) {
			return
		}
	}
}

func (Builder) DownloadPackages(ctx airgap.Context, deps iter.Seq[host.Package]) iter.Seq[container.Step] {
	return func(yield func(container.Step) bool) {
		if !yield(&container.Run{
			Script: bash.StepSeq(func(yield func(bash.Frag) bool) {
				if !yield(bash.Runf(`apt-get install -y %s %s`,
					bash.Fields(
						"--download-only",
						"--reinstall",
						`-o Dir::Cache="/opt/offline-repo"`,
						`-o Dir::Cache::archives="debs"`,
					),
					bash.FieldSeq(func(yield func(string) bool) {
						for dep := range deps {
							if strings.HasSuffix(dep.Name, ".run") {
								continue
							}

							pkg := dep.Name

							if dep.Version != "" {
								pkg += fmt.Sprintf("=$(apt-cache madison %s | grep %q | awk '{print $3}' | head -n 1)", dep.Name, dep.Version)
							}

							if !yield(pkg) {
								return
							}
						}
					}),
				)) {
					return
				}
			}),
		}) {
			return
		}
	}
}

func (b Builder) BuildOfflineRepo(ctx airgap.Context, actions iter.Seq[airgap.Action]) iter.Seq[container.Step] {
	return func(yield func(container.Step) bool) {
		if !yield(&container.Run{
			WorkDir: b.WorkDir(),
			Script: bash.Steps(
				bash.Run("find ./debs/partial/ -name '*.deb' -exec mv -t ./debs/ {} +"),
				bash.Run("rm -r ./debs/partial"),
				bash.Run("dpkg-scanpackages debs /dev/null | gzip -9c > Packages.gz"),
			),
		}) {
			return
		}

		f := bash.FileSeq(func(yield func(bash.Frag) bool) {
			for a := range actions {
				if !yield(a.Run()) {
					return
				}
			}
		})

		if !yield(&container.File{
			Path:     filepath.Join(b.WorkDir(), "offline.sh"),
			Contents: []byte(f.String()),
		}) {
			return
		}
	}
}
