package errors

import "fmt"

type Trace struct {
	File        string   `json:"file"`
	Function    string   `json:"function"`
	Line        int      `json:"line"`
	Annotations []string `json:"annotations"`
}

func (t *Trace) String() string {
	// TODO: Modify to fit how we want the trace to look
	return fmt.Sprintf(
		"%s\n\t%s:%d\n\t%v\n",
		t.File,
		t.Function,
		t.Line,
		t.Annotations,
	)
}
