package errors

import "fmt"

type Trace struct {
	Location   string `json:"location"`
	File       string `json:"file"`
	Function   string `json:"function"`
	Line       int    `json:"line"`
	Annotation string `json:"annotation"`
}

func (t *Trace) String() string {
	return fmt.Sprintf(
		"%s/%s %s @ line %d \n\t %s",
		t.Location,
		t.File,
		t.Function,
		t.Line,
		t.Annotation,
	)
}
