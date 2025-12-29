package bash

import (
	"context"
	"fmt"
	"iter"
	"strings"
)

func lineBreakOrTab(ctx context.Context) string {
	i := IdentFromContext(ctx)
	if i > 0 {
		return "\n" + strings.Repeat("\t", i)
	}
	return "\n"
}

func Run(x string) Frag {
	return RunSeq(func(yield func(string) bool) {
		if !yield(x) {
			return
		}
	})
}

func RunSeq(parts iter.Seq[string]) Frag {
	return func(ctx context.Context) iter.Seq[string] {
		return func(yield func(string) bool) {
			if !yield(lineBreakOrTab(ctx)) {
				return
			}

			for p := range parts {
				if !yield(p) {
					return
				}
			}
		}
	}
}

func Runf(f string, args ...any) Frag {
	return RunSeq(func(yield func(string) bool) {
		if !yield(fmt.Sprintf(f, args...)) {
			return
		}
	})
}

func File(steps ...Frag) Frag {
	return FileSeq(func(yield func(Frag) bool) {
		for _, step := range steps {
			if !yield(step) {
				return
			}
		}
	})
}

func FileSeq(steps iter.Seq[Frag]) Frag {
	return func(ctx context.Context) iter.Seq[string] {
		return func(yield func(string) bool) {
			if !yield("#!/bin/bash") {
				return
			}

			for f := range StepSeq(steps)(ctx) {
				if !yield(f) {
					return
				}
			}
		}
	}
}

func Fields(fields ...string) Frag {
	return FieldSeq(func(yield func(s string) bool) {
		for _, f := range fields {
			if !yield(f) {
				return
			}
		}
	})
}

func FieldSeq(fields iter.Seq[string]) Frag {
	return func(ctx context.Context) iter.Seq[string] {
		return func(yield func(string) bool) {
			i := 0

			for f := range fields {
				if i > 0 {
					if !yield(" ") {
						return
					}
				}

				if !yield(f) {
					return
				}

				i++
			}
		}
	}
}

func Steps(steps ...Frag) Frag {
	return StepSeq(func(yield func(Frag) bool) {
		for _, step := range steps {
			if !yield(step) {
				return
			}
		}
	})
}

func Block(start Frag, body Frag, end Frag) Frag {
	return func(ctx context.Context) iter.Seq[string] {
		return func(yield func(string) bool) {
			for s := range Steps(start)(ctx) {
				if !yield(s) {
					return
				}
			}

			for s := range Steps(body)(WithIdent(ctx, IdentFromContext(ctx)+1)) {
				if !yield(s) {
					return
				}
			}

			for s := range Steps(end)(ctx) {
				if !yield(s) {
					return
				}
			}
		}
	}
}

func StepSeq(steps iter.Seq[Frag]) Frag {
	return func(ctx context.Context) iter.Seq[string] {
		return func(yield func(string) bool) {

			for step := range steps {
				for s := range step(ctx) {
					if !yield(s) {
						return
					}
				}
			}
		}
	}
}
