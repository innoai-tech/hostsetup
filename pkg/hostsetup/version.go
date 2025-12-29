package hostsetup

func WithVersion(version string) Version {
	return Version(version)
}

type Version string

func (k Version) setDep(s *Dep) {
	s.Version = string(k)
}
