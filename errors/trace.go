package errors

import (
	"fmt"
	"strings"
)

type trace struct {
	File        string   `json:"file"`
	Function    string   `json:"function"`
	Line        int      `json:"line"`
	Annotations []string `json:"annotations"`
}

func (t *trace) String() string {
	var b strings.Builder
	for _, a := range t.Annotations {
		b.WriteString("\n\t")
		b.WriteString(a)
	}
	return fmt.Sprintf(
		"%s\n\t%s:%d%v\n",
		t.Function,
		t.File,
		t.Line,
		b.String(),
	)
}
