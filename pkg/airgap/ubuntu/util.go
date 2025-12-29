package ubuntu

import (
	"fmt"

	"github.com/innoai-tech/hostsetup/pkg/bash"
)

func ToDistVersion(version string) string {
	switch version {
	case "24.04":
		return "noble"
	case "22.04":
		return "jammy"
	case "20.04":
		return "focal"
	case "18.04":
		return "bionic"
	case "16.04":
		return "xenial"
	default:
		return ""
	}
}

func SetupOfflineRepo(base string) bash.Frag {
	return bash.Steps(
		bash.WriteFile(`/etc/apt/sources.list.d/offline.list`, []byte(fmt.Sprintf(`
deb [trusted=yes] file://%s ./
`, base))),
		bash.Run("apt-get update"),
	)
}
