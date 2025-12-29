package hostsetup

type DepOption interface {
	setDep(d *Dep)
}
type Dep struct {
	Url     string `json:"url,omitzero"`
	Version string `json:"version,omitzero"`
}

func (d *Dep) Build(options ...DepOption) {
	for _, o := range options {
		o.setDep(d)
	}
}
