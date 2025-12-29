package container

import (
	"iter"

	"dagger.io/dagger"
)

func Of(steps ...Step) iter.Seq[Step] {
	return func(yield func(Step) bool) {
		for _, step := range steps {
			if !yield(step) {
				return
			}
		}
	}
}

func ApplySeq(c *dagger.Container, steps iter.Seq[Step]) *dagger.Container {
	for step := range steps {
		c = step.Apply(c)
	}
	return c
}
