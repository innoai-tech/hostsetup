package openeuler

import (
	"cmp"
	"fmt"
	"strings"
)

type Repo struct {
	ID      string
	Name    string
	BaseURL string

	Include []string
	Exclude []string

	GPGCheck  bool
	GPGKeyURL string

	ModuleHotfixes   string
	InstallOnlyLimit int
	Best             bool

	DisableRepoGPGCheck bool
}

func (r *Repo) String() string {
	s := &strings.Builder{}

	_, _ = fmt.Fprintf(s, "[%s]\n", r.ID)
	_, _ = fmt.Fprintf(s, "name=%s.$basearch\n", cmp.Or(r.Name, r.ID))
	_, _ = fmt.Fprintf(s, "enabled=%s\n", boolToInt(true))

	_, _ = fmt.Fprintf(s, "gpgcheck=%s\n", boolToInt(r.GPGCheck))

	if r.GPGKeyURL != "" {
		_, _ = fmt.Fprintf(s, "gpgkey=%s\n", r.GPGKeyURL)
	}

	if r.BaseURL != "" {
		_, _ = fmt.Fprintf(s, "baseurl=%s\n", r.BaseURL)
	}

	if r.ModuleHotfixes != "" {
		_, _ = fmt.Fprintf(s, "module_hotfixes=%s\n", r.ModuleHotfixes)
	}

	if r.InstallOnlyLimit > 0 {
		_, _ = fmt.Fprintf(s, "installonly_limit=%d\n", r.InstallOnlyLimit)
	}

	if r.Best {
		_, _ = fmt.Fprintf(s, "best=%v\n", boolToConfBool(r.Best))
	}

	if len(r.Include) > 0 {
		_, _ = fmt.Fprintf(s, "includepkgs=%s\n", strings.Join(r.Include, ","))
	}

	if len(r.Exclude) > 0 {
		_, _ = fmt.Fprintf(s, "exclude=%s\n", strings.Join(r.Exclude, ","))
	}

	if r.DisableRepoGPGCheck {
		_, _ = fmt.Fprintf(s, "repo_gpgcheck=false\n")
	}

	return s.String()
}

func boolToInt(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func boolToConfBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
