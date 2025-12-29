package common

import (
	"strings"
)

func VersionMajor(semver string) string {
	return strings.Split(semver, ".")[0]
}
