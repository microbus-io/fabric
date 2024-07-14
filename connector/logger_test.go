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
	"testing"

	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/testarossa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestConnector_Log(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	stderror := stderrors.New("error")

	con := New("log.connector")
	testarossa.False(t, con.IsStarted())

	// No-op when logger is nil, no logs to observe
	testarossa.Nil(t, con.logger)
	con.LogDebug(ctx, "This is a log debug message", log.String("someStr", "some string"))
	con.LogInfo(ctx, "This is a log info message", log.String("someStr", "some string"))
	con.LogWarn(ctx, "This is a log warn message", log.Error(stderror), log.String("someStr", "some string"))
	con.LogError(ctx, "This is a log error message", log.Error(stderror), log.String("someStr", "some string"))

	// Start service to initialize logger
	err := con.Startup()
	testarossa.NoError(t, err)
	defer con.Shutdown()

	// Logger initialized, it can now be observed
	testarossa.NotNil(t, con.logger)

	// Observe the logs to assert expected values
	testCore, observedLogs := observer.New(zap.DebugLevel)
	con.logger = zap.New(zapcore.NewTee(testCore, con.logger.Core()))
	testarossa.Zero(t, observedLogs.Len())

	con.LogDebug(ctx, "This is a log debug message", log.String("someStr", "some string"))
	con.LogInfo(ctx, "This is a log info message", log.String("someStr", "some string"))
	con.LogWarn(ctx, "This is a log warn message", log.Error(stderror), log.String("someStr", "some string"))
	con.LogError(ctx, "This is a log error message", log.Error(stderror), log.String("someStr", "some string"))
	testarossa.Equal(t, 3, observedLogs.Len())

	logs := observedLogs.All()
	testarossa.Equal(t, zap.InfoLevel, logs[0].Level)
	testarossa.Equal(t, "This is a log info message", logs[0].Message)
	testarossa.Equal(t, zap.WarnLevel, logs[1].Level)
	testarossa.Equal(t, "This is a log warn message", logs[1].Message)
	testarossa.Equal(t, zap.ErrorLevel, logs[2].Level)
	testarossa.Equal(t, "This is a log error message", logs[2].Message)
}
