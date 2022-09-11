package connector

import "fmt"

// LogInfo logs a message to standard output
func (c *Connector) LogInfo(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// LogError logs an error to standard output
func (c *Connector) LogError(err error) {
	fmt.Printf("%+v\n", err)
}
