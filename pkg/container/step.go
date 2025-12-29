package container

import (
	"fmt"
	"maps"
	"slices"

	"dagger.io/dagger"

	"github.com/innoai-tech/hostsetup/pkg/bash"
)

type Step interface {
	Apply(c *dagger.Container) *dagger.Container
}

type File struct {
	Path     string
	Source   *dagger.File
	Contents []byte
}

func (f *File) Apply(c *dagger.Container) *dagger.Container {
	if len(f.Contents) > 0 {
		return c.WithNewFile(f.Path, string(f.Contents))
	}
	return c.WithFile(f.Path, f.Source)
}

type Env map[string]string

func (env Env) Apply(c *dagger.Container) *dagger.Container {
	if len(env) == 0 {
		return c
	}

	for _, k := range slices.Sorted(maps.Keys(env)) {
		c = c.WithEnvVariable(k, env[k])
	}

	return c
}

type Run struct {
	WorkDir string
	Env     map[string]string
	Script  bash.Frag
}

func (r *Run) Apply(c *dagger.Container) *dagger.Container {
	if r.WorkDir != "" {
		c = c.WithWorkdir(r.WorkDir)
	}

	c = Env(r.Env).Apply(c)

	return c.WithExec([]string{
		"sh", "-c",
		fmt.Sprintf("%s", r.Script),
	})
}
