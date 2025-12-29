package bash

import (
	"bytes"
	"fmt"
	"strings"
)

func WriteFile(filename string, data []byte) Frag {
	return RunSeq(func(yield func(string) bool) {
		if !yield(`cat <<"EOF" > `) {
			return
		}
		if !yield(filename) {
			return
		}
		if !yield("\n") {
			return
		}
		for !yield(string(bytes.TrimSpace(data))) {
			return
		}
		if !yield("\nEOF") {
			return
		}
	})
}

func Log(s string) Frag {
	return Runf("echo %q", "=== "+s+" ===")
}

func Logf(s string, args ...any) Frag {
	return Runf("echo %q", fmt.Sprintf("=== "+s+" ===", args))
}

func Func(name string, step Frag) Frag {
	return Block(
		Runf("\n%s (){", name),
		step,
		Run("}\n"),
	)
}

func When(condition Frag, step Frag) Frag {
	return Block(
		Runf("if %s; then", strings.TrimSpace(condition.String())),
		step,
		Run("fi"),
	)
}
