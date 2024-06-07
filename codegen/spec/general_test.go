/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpec_General(t *testing.T) {
	t.Parallel()

	var gen General

	err := yaml.Unmarshal([]byte(`
host: super.service
description: foo
`), &gen)
	assert.NoError(t, err)

	err = yaml.Unmarshal([]byte(`
host: $uper.$ervice
description: foo
`), &gen)
	assert.ErrorContains(t, err, "invalid host")

	err = yaml.Unmarshal([]byte(`
host:
description: foo
`), &gen)
	assert.Error(t, err, "invalid host")

	err = yaml.Unmarshal([]byte(`
description: foo
`), &gen)
	assert.Error(t, err, "invalid host")
}
