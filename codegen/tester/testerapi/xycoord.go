/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package testerapi

// XYCoord is a non-primitive type holding X,Y coordinates.
type XYCoord struct {
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`
}
