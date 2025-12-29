package openeuler

import (
	"cmp"
	"fmt"
	"iter"
	"os"
	"path"
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
		Name:    "openeuler",
		Short:   "oe",
		Version: cmp.Or(os.Getenv("TARGET_VERSION"), "22.03"),
		Arch:    cmp.Or(os.Getenv("TARGET_ARCH"), runtime.GOARCH),
	}
}

var _ airgap.WithImageBuilder = Builder{}

func (Builder) BuildImage(ctx airgap.PrepareContext) string {
	parts := strings.Split(ctx.OS().Version, "sp")
	if len(parts) > 1 {
		return fmt.Sprintf("openeuler/openeuler:%s-lts-sp%s", parts[0], parts[1])
	}
	return fmt.Sprintf("openeuler/openeuler:%s-lts", parts[0])
}

func (b Builder) Setup(ctx airgap.Context) iter.Seq[container.Step] {
	return container.Of(
		&container.Run{
			WorkDir: b.WorkDir(),
			Script: bash.Steps(
				bash.Run("dnf makecache"),
				bash.Runf("dnf install -y %s", bash.Fields(
					"wget",
					"createrepo_c",
					"dnf-utils",
					"ca-certificates",
				)),
			),
		},
	)
}

func (b Builder) AddSource(ctx airgap.Context, source host.Source) iter.Seq[container.Step] {
	client := ctx.Client()

	return func(yield func(container.Step) bool) {
		switch path.Ext(source.Url) {
		case ".repo":
			if !yield(&container.File{
				Path:   fmt.Sprintf("/etc/yum.repos.d/%s.repo", source.Name),
				Source: client.HTTP(source.Url),
			}) {
				return
			}
		case ".rpm":
			// local repo
			rpmFilename := fmt.Sprintf("/opt/%s.rpm", source.Name)

			if !yield(&container.File{
				Path:   rpmFilename,
				Source: client.HTTP(source.Url),
			}) {
				return
			}

			if !yield(&container.Run{
				Script: bash.Runf("rpm -i %s", rpmFilename),
			}) {
				return
			}
		default:

		}

		if len(source.Contents) > 0 {
			if !yield(&container.File{
				Path:     fmt.Sprintf("/etc/yum.repos.d/%s.repo", source.Name),
				Contents: source.Contents,
			}) {
				return
			}
		}

		if source.KeyDownloadUrl != "" {
			keyFile := fmt.Sprintf("/etc/pki/rpm-gpg/RPM-GPG-KEY-%s", source.Name)

			if !yield(&container.File{
				Path:   keyFile,
				Source: client.HTTP(source.KeyDownloadUrl),
			}) {
				return
			}

			if !yield(&container.Run{
				Script: bash.Runf("rpm --import %s", keyFile),
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
					Path:   fmt.Sprintf("/opt/temp-repo/rpms/%s", path.Base(dep.DownloadURL)),
					Source: client.HTTP(dep.DownloadURL),
				}) {
					return
				}

				hasTempRepo = true
			}
		}

		if hasTempRepo {
			if !yield(&container.File{
				Path: "/etc/yum.repos.d/temp.repo",
				Contents: []byte((&Repo{
					ID:      "temp-repo",
					BaseURL: "file:///opt/temp-repo/rpms",
				}).String()),
			}) {
				return
			}

			if !yield(&container.Run{
				WorkDir: "/opt/temp-repo",
				Script:  bash.Run("createrepo_c ./rpms"),
			}) {
				return
			}
		}

		if !yield(&container.Run{
			WorkDir: b.WorkDir(),
			Script:  bash.Run("dnf clean all"),
		}) {
			return
		}
	}
}

func (b Builder) DownloadPackages(ctx airgap.Context, deps iter.Seq[host.Package]) iter.Seq[container.Step] {
	return func(yield func(container.Step) bool) {
		if !yield(&container.Run{
			Script: bash.Runf(`dnf install --downloadonly --setopt=install_weak_deps=False -y --destdir=%s %s`,
				path.Join(b.WorkDir(), "rpms"),
				bash.FieldSeq(func(yield func(string) bool) {
					for dep := range deps {
						if strings.HasSuffix(dep.Name, ".run") {
							continue
						}

						rpmFullName := dep.Name

						if dep.Version != "" {
							rpmFullName += fmt.Sprintf("-%s", dep.Version)
						}

						if !yield(rpmFullName) {
							return
						}
					}
				}),
			),
		}) {
			return
		}
	}
}

func (b Builder) BuildOfflineRepo(ctx airgap.Context, actions iter.Seq[airgap.Action]) iter.Seq[container.Step] {
	return func(yield func(container.Step) bool) {
		if !yield(&container.Run{
			WorkDir: b.WorkDir(),
			Script:  bash.Run("createrepo_c ./rpms"),
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
			Path:     path.Join(b.WorkDir(), "offline.sh"),
			Contents: []byte(f.String()),
		}) {
			return
		}
	}
}
