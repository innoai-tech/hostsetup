package airgap

import (
	"iter"
	"slices"

	"github.com/innoai-tech/hostsetup/pkg/bash"
	"github.com/innoai-tech/hostsetup/pkg/host"
)

type Action interface {
	Run() bash.Frag
}

type WithDeps interface {
	Deps(ctx Context) iter.Seq[host.Package]
}

func Func(name string, actions ...Action) Action {
	return &funcAction{
		name:    name,
		actions: actions,
	}
}

func FuncSeq(name string, actions iter.Seq[Action]) Action {
	return &funcAction{
		name:    name,
		actions: slices.Collect(actions),
	}
}

type funcAction struct {
	name    string
	actions []Action
}

func (a *funcAction) Run() bash.Frag {
	return bash.Func(a.name, bash.StepSeq(func(yield func(bash.Frag) bool) {
		for _, action := range a.actions {
			if !yield(action.Run()) {
				return
			}
		}
	}))
}

func (a *funcAction) Deps(ctx Context) iter.Seq[host.Package] {
	return func(yield func(host.Package) bool) {
		for _, action := range a.actions {
			if deps, ok := action.(WithDeps); ok {
				for d := range deps.Deps(ctx) {
					if !yield(d) {
						return
					}
				}
			}
		}
	}
}

func When(when bash.Frag, actions ...Action) Action {
	return &conditionAction{
		when:    when,
		actions: actions,
	}
}

type conditionAction struct {
	when    bash.Frag
	actions []Action
}

func (a *conditionAction) Run() bash.Frag {
	return bash.When(a.when, bash.StepSeq(func(yield func(bash.Frag) bool) {
		for _, action := range a.actions {
			if !yield(action.Run()) {
				return
			}
		}
	}))
}

func (a *conditionAction) Deps(ctx Context) iter.Seq[host.Package] {
	return func(yield func(host.Package) bool) {
		for _, action := range a.actions {
			if deps, ok := action.(WithDeps); ok {
				for d := range deps.Deps(ctx) {
					if !yield(d) {
						return
					}
				}
			}
		}
	}
}

func StepSeq(steps iter.Seq[bash.Frag]) Action {
	return actionGroup(slices.Collect(steps))
}

func Steps(steps ...bash.Frag) Action {
	return actionGroup(steps)
}

type actionGroup []bash.Frag

func (x actionGroup) Run() bash.Frag {
	return bash.Steps(x...)
}
