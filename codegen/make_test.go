package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodegen_FindReplaceReturnedErrors(t *testing.T) {
	t.Parallel()

	testCases := []string{
		`
return err
`, `
return errors.Trace(err)
`,

		`
	return err
`, `
	return errors.Trace(err)
`,

		`
	return 1, map[string]bool{}, err
`, `
	return 1, map[string]bool{}, errors.Trace(err)
`,

		`
	return err // No trace
`, `
	return err // No trace
`,

		`
	if err := doSomething(); err!=nil {
		return err
	}
`, `
	if err := doSomething(); err!=nil {
		return errors.Trace(err)
	}
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		modified := findReplaceReturnedErrors(testCases[i])
		assert.Equal(t, testCases[i+1], modified)
	}
}

func TestCodegen_FindReplaceImportErrors(t *testing.T) {
	t.Parallel()

	testCases := []string{
		`
import "errors"
`, `
import "github.com/microbus-io/fabric/errors"
`,

		`
import (
	"errors"
	"fmt"
)
`, `
import (
	"fmt"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
	"errors"
)
`, `
import (
	"fmt"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
	"errors"
	"net"
)
`, `
import (
	"fmt"
	"net"

	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"errors"
)
`, `
import (
	"github.com/microbus-io/fabric/errors"
)
`,

		`
import (
	"fmt"
)
`, `
import (
	"fmt"
)
`,

		`
import "fmt"
`, `
import "fmt"
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		modified := findReplaceImportErrors(testCases[i])
		assert.Equal(t, testCases[i+1], modified, "test case %d", (i/2)+1)
	}
}
