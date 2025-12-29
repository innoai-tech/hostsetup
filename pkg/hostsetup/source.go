package hostsetup

type Source struct {
	Url    string `json:"url,omitzero"`
	GPGKey string `json:"gpgKey,omitzero"`
}

func (s *Source) Build(options ...SourceOption) {
	for _, o := range options {
		o.setSource(s)
	}
}

func WithURL(u string, args ...any) *URL {
	if len(args) > 0 {
		switch x := args[0].(type) {
		case map[string]any:
			return &URL{
				url:    u,
				values: x,
			}
		}
	}

	return &URL{
		url: u,
	}
}

type SourceOption interface {
	setSource(s *Source)
}

func WithGPGKey(u string) GPGKey {
	return GPGKey(u)
}

type GPGKey string

func (k GPGKey) setSource(s *Source) {
	s.GPGKey = string(k)
}
