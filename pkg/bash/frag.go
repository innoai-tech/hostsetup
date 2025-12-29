package bash

import (
	"context"
	"iter"
	"strings"
)

type Frag func(ctx context.Context) iter.Seq[string]

func (f Frag) String() string {
	ctx := context.Background()
	b := &strings.Builder{}
	for p := range f(ctx) {
		b.WriteString(p)
	}
	return b.String()
}

type contextIdent struct{}

func WithIdent(pctx context.Context, i int) context.Context {
	return context.WithValue(pctx, contextIdent{}, i)
}

func IdentFromContext(ctx context.Context) int {
	if v, ok := ctx.Value(contextIdent{}).(int); ok {
		return v
	}
	return 0
}
