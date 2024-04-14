/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import "time"

// callback holds settings for a user callback handler, such as the OnStartup and OnShutdown callbacks.
type callback struct {
	Name     string
	Handler  any
	Interval time.Duration
	Ticker   *time.Ticker
}
