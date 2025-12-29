package main

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/innoai-tech/hostsetup/pkg/hostsetup"
	"github.com/innoai-tech/hostsetup/pkg/hostsetup/ubuntu"
)

func main() {
	u := &ubuntu.Ubuntu{
		OS: hostsetup.OS{
			Name:    "ubuntu",
			Version: cmp.Or(os.Getenv("TARGET_VERSION"), "22.04"),
			Arch:    cmp.Or(os.Getenv("TARGET_ARCH"), runtime.GOARCH),
		},
		Vars: map[string]string{
			"NVIDIA_DRIVER_VERSION": "580.95.05",
		},
	}

	if u.OS.Version == "18.04" {
		if u.OS.Arch == "arm64" {
			panic("ubuntu18.04 arm64 is not unsupported")
		}

		u.Vars = map[string]string{
			"NVIDIA_DRIVER_VERSION": "525.147.05",
		}

		u.AddSource(
			"cuda-ubuntu1804",
			hostsetup.WithURL("https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/"),
			hostsetup.WithGPGKey("https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/3bf863cc.pub"),
		)
	}

	u.AddSource(
		"nvidia-container-toolkit",
		hostsetup.WithURL("https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list"),
		hostsetup.WithGPGKey("https://nvidia.github.io/libnvidia-container/gpgkey"),
	)

	// kernel
	if u.OS.Version == "18.04" || u.OS.Version == "20.04" {
		u.AddDep(fmt.Sprintf("linux-image-generic-hwe-%s", u.OS.Version))
	}

	u.AddDep("curl")
	u.AddDep("dkms")
	u.AddDep("dpkg-dev")
	u.AddDep("xfsprogs")

	// nvidia driver
	u.AddDep(
		fmt.Sprintf("nvidia-headless-%s", strings.Split(u.Vars["NVIDIA_DRIVER_VERSION"], ".")[0]),
		hostsetup.WithVersion(u.Vars["NVIDIA_DRIVER_VERSION"]),
	)
	u.AddDep(
		fmt.Sprintf("nvidia-utils-%s", strings.Split(u.Vars["NVIDIA_DRIVER_VERSION"], ".")[0]),
		hostsetup.WithVersion(u.Vars["NVIDIA_DRIVER_VERSION"]),
	)

	// nvidia container toolkit
	u.AddDep("nvidia-container-toolkit")

	// nftables 1.0.1
	if u.OS.Version == "18.04" || u.OS.Version == "20.04" {
		u.AddDep("nftables-custom", hostsetup.WithURL(
			"{{ .base_url }}/nftables-custom_1.0.1-1-{{ .release }}_{{ .arch }}.deb",
			map[string]any{
				"base_url": "https://github.com/innoai-tech/nftables/releases/download/latest",
				"release":  u.OS.Release(),
				"arch":     u.OS.Arch,
			},
		))
	}

	if u.OS.Version == "18.04" {
		u.AddDep("iptables-nftables-compat")
	}

	if err := u.Build(context.Background(), "./target"); err != nil {
		panic(err)
	}
}
