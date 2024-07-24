/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connector

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestConnector_Log(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	stderror := stderrors.New("error")

	con := New("log.connector")
	testarossa.False(t, con.IsStarted())

	// No-op when logger is nil, no logs to observe
	testarossa.Nil(t, con.logger)
	con.LogDebug(ctx, "This is a log debug message", "someStr", "some string")
	con.LogInfo(ctx, "This is a log info message", "someStr", "some string")
	con.LogWarn(ctx, "This is a log warn message", "error", stderror, "someStr", "some string")
	con.LogError(ctx, "This is a log error message", "error", stderror, "someStr", "some string")

	// Start service to initialize logger
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Logger initialized, it can now be observed
	testarossa.NotNil(t, con.logger)

	// Observe the logs to assert expected values
	var buf strings.Builder
	con.logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	con.LogDebug(ctx, "This is a log debug message", "someStr", "some string")
	con.LogInfo(ctx, "This is a log info message", "someStr", "some string")
	con.LogWarn(ctx, "This is a log warn message", "error", stderror, "someStr", "some string")
	con.LogError(ctx, "This is a log error message", "error", stderror, "someStr", "some string")

	bufStr := buf.String()
	fmt.Println(bufStr)
	testarossa.Contains(t, bufStr, `level=INFO msg="This is a log info message"`)
	testarossa.Contains(t, bufStr, `level=WARN msg="This is a log warn message"`)
	testarossa.Contains(t, bufStr, `level=ERROR msg="This is a log error message"`)
	testarossa.NotContains(t, bufStr, `level=DEBUG msg="This is a log debug message"`)
}
