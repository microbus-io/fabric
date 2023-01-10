// Code generated by Microbus. DO NOT EDIT.

package directoryapi

import (
    "time"
    
    import1_clock "github.com/microbus-io/fabric/clock"
)

var (
    _ time.Duration
)

/*
NullTime refers to github.com/microbus-io/fabric/clock/NullTime.
*/
type NullTime = import1_clock.NullTime

/*
PersonKey is the primary key of the person.
*/
type PersonKey struct {
    Seq int `json:"seq"`
}

/*
Person is a personal record that is registered in the directory.
First and last name and email are required. Birthday is optional.
*/
type Person struct {
    Birthday NullTime `json:"birthday"`
    Email string `json:"email"`
    FirstName string `json:"firstName"`
    Key PersonKey `json:"key"`
    LastName string `json:"lastName"`
}
