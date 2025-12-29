package common

import (
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/innoai-tech/hostsetup/pkg/host"
)

import (
	_ "embed"
)

//go:embed toolchain.toml
var preset []byte

var ToolchainPreset Toolchain

func init() {
	if err := toml.Unmarshal(preset, &ToolchainPreset); err != nil {
		panic(err)
	}
}

type Toolchain struct {
	Deps map[string]Dep `toml:"deps"`
}

func (t Toolchain) ToPackages(release string) iter.Seq[host.Package] {
	return func(yield func(host.Package) bool) {
		for _, pkgName := range slices.Sorted(maps.Keys(t.Deps)) {
			pkg := t.Deps[pkgName]
			if pkg.OS != "" && !strings.HasPrefix(release, pkg.OS) {
				continue
			}

			if match(pkg.Skip, release) {
				continue
			}

			if !yield(host.Package{
				Name: pkgName,
			}) {
				return
			}
		}
	}
}

func match(patterns []string, target string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, p := range patterns {
		// 如果模式以 * 结尾，进行前缀匹配 (如 "oe*" 匹配 "oe2203")
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(target, prefix) {
				return true
			}
		} else {
			// 否则进行精确全匹配 (如 "ubuntu1804")
			if p == target {
				return true
			}
		}
	}

	return false
}

type Dep struct {
	Version string   `toml:"version,omitzero"`
	OS      string   `toml:"os,omitzero"`
	Skip    []string `toml:"skip,omitzero"`
}
