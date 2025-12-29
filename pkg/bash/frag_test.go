package bash

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	file := File(
		Func("hello",
			Runf("echo %q", "1"),
		),

		When(
			Runf(`lspci | grep -iq %q`, "NVIDIA"),
			Steps(
				Runf(`echo %q`, "检测到 NVIDIA 显卡，开始安装驱动..."),
				Runf(`apt-get install -y %s`,
					Fields("nvidia-headless-580", "nvidia-utils-580"),
				),
				Runf(`apt-get install -y %s`,
					Fields("nvidia-container-toolkit"),
				),
			),
		),

		When(
			Run(`[ ! -e /sys/fs/cgroup/cgroup.controllers ]`),
			Steps(
				When(
					Run(`! grep -q "systemd.unified_cgroup_hierarchy=1" /etc/default/grub`),
					Steps(
						Run(`sed -i 's/GRUB_CMDLINE_LINUX_DEFAULT="/&systemd.unified_cgroup_hierarchy=1 /' /etc/default/grub`),
						Run(`update-grub`),
					),
				),
			),
		),

		Func("config-x", WriteFile("/etc/protected.conf", []byte(`x=1
`))),
	)

	fmt.Println(file.String())
}
