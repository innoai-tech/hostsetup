package main

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/innoai-tech/hostsetup/pkg/airgap"
	airgapbuild "github.com/innoai-tech/hostsetup/pkg/airgap/build"
	"github.com/innoai-tech/hostsetup/pkg/airgap/common"
	"github.com/innoai-tech/hostsetup/pkg/airgap/ubuntu"
	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type hostSetup struct {
	ubuntu.Builder
}

func (hostSetup) Supported(ctx airgap.PrepareContext) bool {
	if o := ctx.OS(); o.Version == "18.04" && o.Arch == "arm64" {
		return false
	}
	return true
}

func nvidiaDriverVersion(osVer string) string {
	if strings.HasPrefix(osVer, "18.") || strings.HasPrefix(osVer, "20.") {
		return "535.288.01"
	}
	if strings.HasPrefix(osVer, "22.") {
		return "550.163.01"
	}
	return "580.126.09"
}

func (hostSetup) BuildVars(ctx airgap.PrepareContext) map[string]string {
	return map[string]string{
		"NVIDIA_DRIVER_VERSION": nvidiaDriverVersion(ctx.OS().Version),
	}
}

var _ airgap.WithSources = &hostSetup{}

func (hostSetup) Sources(ctx airgap.Context) iter.Seq[host.Source] {
	return func(yield func(host.Source) bool) {
		if ctx.OS().Version == "18.04" {
			if !yield(host.Source{
				Name: "cuda-ubuntu1804",
				Contents: []byte(`deb https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/ /
`),
				KeyDownloadUrl: "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/3bf863cc.pub",
			}) {
				return
			}
		}

		if !yield(host.Source{
			Name:           "nvidia-container-toolkit",
			Url:            "https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list",
			KeyDownloadUrl: "https://nvidia.github.io/libnvidia-container/gpgkey",
		}) {
			return
		}

		if !yield(host.Source{
			Name:           "docker",
			KeyType:        "asc",
			KeyDownloadUrl: "https://download.docker.com/linux/ubuntu/gpg",
			Contents: []byte(fmt.Sprintf(`deb [arch=%s] https://download.docker.com/linux/ubuntu %s stable
`, ctx.OS().Arch, ubuntu.ToDistVersion(ctx.OS().Version))),
		}) {
			return
		}
	}
}

var _ airgap.WithActions = &hostSetup{}

func (hostSetup) Actions(ctx airgap.Context) iter.Seq[airgap.Action] {
	return func(yield func(airgap.Action) bool) {
		o := ctx.OS()

		if !yield(airgap.Func(
			common.INIT,
			airgap.Steps(
				ubuntu.SetupOfflineRepo("/opt/offline-repo"),
			),
		)) {
			return
		}

		if o := ctx.OS(); o.Version == "18.04" || o.Version == "20.04" {
			if !yield(
				airgap.Func(
					common.UPDATE_KERNEL,
					ubuntu.InstallPackages([]host.Package{
						{Name: fmt.Sprintf("linux-image-generic-hwe-%s", o.Version)},
						{Name: fmt.Sprintf("linux-headers-generic-hwe-%s", o.Version)},
					}),
				),
			) {
				return
			}
		}

		if !yield(airgap.Func(
			common.UPDATE_CGROUP,
			airgap.When(
				bash.Runf(`[ ! -e /sys/fs/cgroup/cgroup.controllers ]`),
				airgap.Steps(bash.When(
					bash.Run(`! grep -q "systemd.unified_cgroup_hierarchy=1" /etc/default/grub`),
					bash.Steps(
						bash.Log("配置 cgroup v2"),
						bash.Run(`sed -i 's/GRUB_CMDLINE_LINUX_DEFAULT="/&systemd.unified_cgroup_hierarchy=1 /' /etc/default/grub`),
						bash.Run(`update-grub`),
						bash.Log(`已完成. 重启系统后生效`),
					),
				)),
			),
		)) {
			return
		}

		if !yield(airgap.Func(common.PRUNE_NOUVEAU, airgap.Steps(
			bash.When(
				bash.Run(`lsmod | grep -q "nouveau"`),
				bash.Steps(
					bash.WriteFile("/etc/modprobe.d/blacklist-nouveau.conf", []byte(`
blacklist nouveau
options nouveau modeset=0
`)),
					bash.Run("update-initramfs -u"),
					bash.Log("重启后生效"),
				),
			),
		))) {
			return
		}

		if !yield(airgap.FuncSeq(common.INSTALL_NVIDIA_DRIVER, func(yield func(airgap.Action) bool) {
			if !yield(airgap.When(
				bash.Runf(`lspci | grep -iq %q`, "NVIDIA"),
				// 驱动依赖
				ubuntu.InstallPackages([]host.Package{
					{Name: "dkms"},
					{Name: "build-essential"},
					{Name: "linux-headers-generic"},
					{Name: fmt.Sprintf("linux-headers-generic-hwe-%s", o.Version), DownloadOnly: true},
				}),
				// 驱动
				ubuntu.InstallPackages([]host.Package{
					{
						Name: fmt.Sprintf("NVIDIA-Linux-%s-%s.run", o.GNUArch(), ctx.Var("NVIDIA_DRIVER_VERSION")),
						DownloadURL: func() string {
							if o.Arch == "amd64" {
								return fmt.Sprintf(
									"https://us.download.nvidia.com/XFree86/Linux-x86_64/%[1]s/NVIDIA-Linux-x86_64-%[1]s.run",
									ctx.Var("NVIDIA_DRIVER_VERSION"),
								)
							}
							return fmt.Sprintf(
								"https://us.download.nvidia.com/tesla/%[1]s/NVIDIA-Linux-aarch64-%[1]s.run",
								ctx.Var("NVIDIA_DRIVER_VERSION"),
							)
						}(),
						Args: []string{
							"--no-questions",
							"--accept-license",
							"--no-opengl-files",
							"--no-nouveau-check",
							"--no-x-check",
							"--dkms",
							"--silent",
						},
					},
				}),
				ubuntu.InstallPackages([]host.Package{
					{Name: "nvidia-container-toolkit"},
				}),
			)) {
				return
			}
		}),
		) {
			return
		}

		if !yield(airgap.Func(
			common.INSTALL_DOCKER,
			ubuntu.InstallPackages([]host.Package{
				{Name: "docker-ce"},
				{Name: "docker-ce-cli"},
				{Name: "containerd.io"},
				{Name: "docker-compose-plugin"},
			}),
		)) {
			return
		}

		if !yield(airgap.Func(common.INSTALL_TOOLCHAIN, ubuntu.InstallPackages(
			slices.Collect(common.ToolchainPreset.ToPackages(ctx.OS().DistroRelease())),
		))) {
			return
		}

		if !yield(airgap.FuncSeq(common.SWITCH_IPTABLES_NFT, func(yield func(airgap.Action) bool) {
			if o.Version == "18.04" {
				if !yield(ubuntu.InstallPackages([]host.Package{
					{
						Name: "nftables-custom",
						DownloadURL: fmt.Sprintf(
							"%s/nftables-custom_1.0.1-1-%s_%s.deb",
							"https://github.com/innoai-tech/nftables/releases/download/latest",
							o.DistroRelease(),
							o.Arch,
						),
					},
				})) {
					return
				}

				if !yield(ubuntu.InstallPackages([]host.Package{
					{Name: "iptables-nftables-compat"},
				})) {
					return
				}
			}

			if !yield(airgap.When(
				bash.Runf("iptables -V | grep -iq %s", "legacy"),
				airgap.Steps(
					bash.Run("update-alternatives --set iptables /usr/sbin/iptables-nft"),
					bash.Run("update-alternatives --set ip6tables /usr/sbin/ip6tables-nft"),
				),
			)) {
				return
			}
		})) {
			return
		}
	}
}

func main() {
	if err := airgapbuild.Build(context.Background(), &hostSetup{}, "./target"); err != nil {
		panic(err)
	}
}
