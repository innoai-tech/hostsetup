package openeuler

import "github.com/innoai-tech/hostsetup/pkg/hostsetup"

type OpenEuler struct {
	Vars    map[string]string           `json:"vars"`
	OS      hostsetup.OS                `json:"os"`
	Sources map[string]hostsetup.Source `json:"sources"`
	Deps    map[string]hostsetup.Dep    `json:"deps"`
}

func (u *OpenEuler) AddSource(name string, options ...hostsetup.SourceOption) {
	if u.Sources == nil {
		u.Sources = map[string]hostsetup.Source{}
	}

	source := hostsetup.Source{}
	source.Build(options...)

	u.Sources[name] = source
}

func (u *OpenEuler) AddDep(name string, options ...hostsetup.DepOption) {
	if u.Deps == nil {
		u.Deps = map[string]hostsetup.Dep{}
	}

	dep := hostsetup.Dep{}
	dep.Build(options...)

	u.Deps[name] = dep
}
