package main

import (
	"cmp"
	"context"
	"os"
	"runtime"

	"github.com/innoai-tech/hostsetup/pkg/hostsetup"
	"github.com/innoai-tech/hostsetup/pkg/hostsetup/openeuler"
)

func main() {
	u := &openeuler.OpenEuler{
		OS: hostsetup.OS{
			Name:    "openeuler",
			Short:   "oe",
			Version: cmp.Or(os.Getenv("TARGET_VERSION"), "22.03"),
			Arch:    cmp.Or(os.Getenv("TARGET_ARCH"), runtime.GOARCH),
		},
	}

	// nftables 1.0.1
	if u.OS.Version == "22.03" {
		u.AddDep("nftables-custom", hostsetup.WithURL(
			"{{ .base_url }}/nftables-custom-1.0.1-1.{{ .release }}.{{ .arch }}.rpm",
			map[string]any{
				"base_url": "https://github.com/innoai-tech/nftables/releases/download/latest",
				"release":  u.OS.Release(),
				"arch":     u.OS.GNUArch(),
			},
		))
	}

	// 310 npu driver & mcu
	u.AddDep("A300-3010-npu-driver",
		hostsetup.WithURL(
			"{{ .base_url }}/A300-3010-npu-driver-24.1.0-1.{{ .arch }}.rpm",
			map[string]any{
				"base_url": "https://github.com/innoai-tech/ascend-toolkit/releases/download/latest",
				"arch":     u.OS.GNUArch(),
			},
		))

	u.AddDep("A300-3010-npu-firmware",
		hostsetup.WithURL(
			"{{ .base_url }}/A300-3010-npu-firmware-7.5.0.2.220-1.noarch.rpm",
			map[string]any{
				"base_url": "https://github.com/innoai-tech/ascend-toolkit/releases/download/latest",
			},
		))

	if err := u.Build(context.Background(), "./target"); err != nil {
		panic(err)
	}
}
