/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package trc

import "context"

// Tracer is an interface that allows starting a new trace
type Tracer interface {
	StartSpan(ctx context.Context, spanName string, opts ...Option) (context.Context, Span)
}
