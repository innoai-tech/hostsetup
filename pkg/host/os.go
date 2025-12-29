package host

import (
	"cmp"
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

func (o OS) Distro() string {
	return cmp.Or(o.Short, o.Name)
}

func (o OS) DistroRelease() string {
	return o.Distro() + strings.ReplaceAll(o.Version, ".", "")
}
