package hostsetup

import (
	"cmp"
	"fmt"
	"strings"
)

type OS struct {
	Name    string `json:"name"`
	Short   string `json:"short,omitzero"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

func (o OS) GNUArch() string {
	switch o.Arch {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	}
	return "noarch"
}

func (o OS) Release() string {
	return fmt.Sprintf("%s%s", cmp.Or(o.Short, o.Name), strings.ReplaceAll(o.Version, ".", ""))
}
