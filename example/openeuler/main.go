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
	"github.com/innoai-tech/hostsetup/pkg/airgap/openeuler"
	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type hostSetup struct {
	openeuler.Builder
}

var _ airgap.WithSources = &hostSetup{}

func (hostSetup) Supported(ctx airgap.PrepareContext) bool {
	return true
}

func (hostSetup) BuildVars(ctx airgap.PrepareContext) map[string]string {
	if strings.HasPrefix(ctx.OS().Version, "22.") {
		return map[string]string{
			"NVIDIA_DRIVER_VERSION": "535.288.01",
		}
	}

	return map[string]string{
		"NVIDIA_DRIVER_VERSION": "550.163.01",
	}
}

func (hostSetup) Sources(ctx airgap.Context) iter.Seq[host.Source] {
	return func(yield func(host.Source) bool) {
		o := ctx.OS()

		rhelVer := openeuler.ToRhelVersion(o)

		if !yield(host.Source{
			Name: "docker-ce",
			Contents: []byte((&openeuler.Repo{
				ID:        "docker-ce-stable",
				BaseURL:   fmt.Sprintf("https://download.docker.com/linux/rhel/%s/$basearch/stable", rhelVer),
				GPGKeyURL: "https://download.docker.com/linux/centos/gpg",
				GPGCheck:  true,
				Exclude: []string{
					"*.i686",
				},
			}).String()),
		}) {
			return
		}

		if !yield(host.Source{
			Name: "nvidia-container-toolkit",
			Contents: []byte((&openeuler.Repo{
				ID:        "nvidia-container-toolkit",
				BaseURL:   "https://nvidia.github.io/libnvidia-container/stable/rpm/$basearch",
				GPGKeyURL: "https://nvidia.github.io/libnvidia-container/gpgkey",
				GPGCheck:  true,
				Exclude: []string{
					"*.i686",
				},
				DisableRepoGPGCheck: true,
			}).String()),
		}) {
			return
		}
	}
}

var _ airgap.WithActions = &hostSetup{}

func (s hostSetup) Actions(ctx airgap.Context) iter.Seq[airgap.Action] {
	return func(yield func(airgap.Action) bool) {
		o := ctx.OS()

		if !yield(airgap.Func(common.INIT,
			airgap.Steps(
				openeuler.SetupOfflineRepo("/opt/offline-repo"),
			),
		)) {
			return
		}

		if !yield(airgap.Func(common.UPDATE_CGROUP,
			airgap.When(
				bash.Runf(`[ ! -e /sys/fs/cgroup/cgroup.controllers ]`),
				airgap.Steps(
					bash.Log("配置 cgroup v2"),
					bash.Run(`grubby --update-kernel=ALL --args="systemd.unified_cgroup_hierarchy=1"`),
					bash.Run(`grubby --args="cgroup_no_v1=all" --update-kernel="/boot/vmlinuz-$(uname -r)"`),
					bash.Run(`grub2-mkconfig -o /boot/grub2/grub.cfg`),
					bash.Log("已完成. 重启系统后生效"),
				),
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
					bash.Run("dracut --force"),
					bash.Log("重启后生效"),
				),
			),
		))) {
			return
		}

		if !yield(airgap.Func(common.INSTALL_NVIDIA_DRIVER,
			airgap.When(
				bash.Runf(`lspci | grep -iq %q`, "NVIDIA"),
				openeuler.InstallPackages([]host.Package{
					// 驱动依赖
					{Name: "kernel-devel"}, // OpenEuler 内核一般不可升级
					{Name: "kernel-headers"},
					{Name: "gcc"},
					{Name: "make"},
					{Name: "dkms"},
					{Name: "elfutils-libelf-devel"},
					// 驱动
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
							"--no-opengl-files",
							"--no-nouveau-check",
							"--no-cc-version-check", // 忽略编译器版本微调冲突
							"--dkms",                // 强烈建议开启，内核升级后驱动会自动重编
							"--no-x-check",          // Headless 环境必备
							"--utility-prefix=/usr", // 确保 nvidia-smi 等工具放在标准路径
							"--no-questions",        // 自动回答所有问题
							"--accept-license",      // 自动接受协议
							"--silent",
						},
					},
				}),
				openeuler.InstallPackages([]host.Package{
					// 容器运行时
					{Name: "nvidia-container-toolkit"},
				}),
			),
		)) {
			return
		}

		if !yield(airgap.Func(common.INSTALL_ASCEND_DRIVER, openeuler.InstallPackages([]host.Package{
			{
				Name: "A300-3010-npu-driver",
				DownloadURL: fmt.Sprintf(
					"%s/A300-3010-npu-driver-24.1.0-1.%s.rpm",
					"https://github.com/innoai-tech/ascend-toolkit/releases/download/latest",
					ctx.OS().GNUArch(),
				),
			},
			{
				Name: "A300-3010-npu-firmware",
				DownloadURL: fmt.Sprintf(
					"%s/A300-3010-npu-firmware-7.5.0.2.220-1.noarch.rpm",
					"https://github.com/innoai-tech/ascend-toolkit/releases/download/latest",
				),
			},
			{
				Name: fmt.Sprintf(
					"%s/Ascend-docker-runtime_7.1.RC1_linux-%s.run",
					s.WorkDir(),
					ctx.OS().GNUArch(),
				),
				DownloadURL: fmt.Sprintf(
					"%s/Ascend-docker-runtime_7.1.RC1_linux-%s.run",
					"https://github.com/innoai-tech/ascend-toolkit/releases/download/latest",
					ctx.OS().GNUArch(),
				),
				Args: []string{
					"--install",
				},
			},
		}))) {
			return
		}

		if !yield(airgap.Func(common.INSTALL_DOCKER,
			// 冲突卸载
			airgap.Steps(
				bash.Run("dnf remove -y iSulad docker-engine"),
			),
			openeuler.InstallPackages([]host.Package{
				{Name: "docker-ce"},
				{Name: "docker-ce-cli"},
				{Name: "containerd.io"},
				{Name: "docker-compose-plugin"},
			})),
		) {
			return
		}

		if !yield(airgap.Func(common.INSTALL_TOOLCHAIN, openeuler.InstallPackages(
			slices.Collect(common.ToolchainPreset.ToPackages(ctx.OS().DistroRelease())),
		))) {
			return
		}

		if !yield(airgap.FuncSeq(common.SWITCH_IPTABLES_NFT, func(yield func(airgap.Action) bool) {
			if o.Version == "22.03" {
				if !yield(openeuler.InstallPackages([]host.Package{
					{
						Name: "nftables-custom",
						DownloadURL: fmt.Sprintf(
							"%s/nftables-custom-1.0.1-1.%s.%s.rpm",
							"https://github.com/innoai-tech/nftables/releases/download/latest",
							o.DistroRelease(),
							o.GNUArch(),
						),
					},
				})) {
					return
				}
			}

			if !yield(airgap.When(
				bash.Runf("iptables -V | grep -iq %s", "legacy"),
				airgap.Steps(
					// ipv4
					bash.Run("update-alternatives --install /usr/sbin/iptables iptables /usr/sbin/xtables-nft-multi 100"),
					bash.Run("update-alternatives --set iptables /usr/sbin/xtables-nft-multi"),

					// ipv6
					bash.Run("update-alternatives --install /usr/sbin/ip6tables ip6tables /usr/sbin/xtables-nft-multi 100"),
					bash.Run("update-alternatives --set ip6tables /usr/sbin/xtables-nft-multi"),
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
