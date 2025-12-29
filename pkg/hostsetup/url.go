package hostsetup

import (
	"strings"
	"sync"
	"text/template"
)

type URL struct {
	url    string
	values map[string]any

	t   *template.Template
	err error
	o   sync.Once
}

func (u *URL) template() (*template.Template, error) {
	u.o.Do(func() {
		u.t, u.err = template.New(u.url).Parse(u.url)
	})
	return u.t, u.err
}

func (u *URL) toURL() string {
	if u.values != nil {
		t, err := u.template()
		if err != nil {
			panic(err)
		}

		b := &strings.Builder{}
		if err := t.Execute(b, u.values); err != nil {
			panic(err)
		}
		return b.String()
	}
	return u.url
}

func (u *URL) setSource(s *Source) {
	s.Url = u.toURL()
}

func (u *URL) setDep(s *Dep) {
	s.Url = u.toURL()
}
