package openeuler

import (
	"fmt"

	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

func SetupOfflineRepo(base string) bash.Frag {
	return bash.Steps(
		bash.WriteFile("/etc/yum.repos.d/offline.repo", []byte((&Repo{
			ID:      "offline-repo",
			BaseURL: fmt.Sprintf("file://%s/rpms", base),
		}).String())),
		bash.Run("dnf makecache"),
	)
}

func ToRhelVersion(os host.OS) string {
	switch os.Version {
	case "24.03":
		return "9"
	}
	return "8"
}
