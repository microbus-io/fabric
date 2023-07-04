/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package directoryapi

import (
	"github.com/microbus-io/fabric/clock"
)

// PersonKey is the primary key of the person.
type PersonKey struct {
	Seq int `json:"seq,omitempty"`
}

// Person is a personal record that is registered in the directory.
// First and last name and email are required. Birthday is optional.
type Person struct {
	Birthday  clock.NullTime `json:"birthday,omitempty"`
	Email     string         `json:"email,omitempty"`
	FirstName string         `json:"firstName,omitempty"`
	Key       PersonKey      `json:"key,omitempty"`
	LastName  string         `json:"lastName,omitempty"`
}
