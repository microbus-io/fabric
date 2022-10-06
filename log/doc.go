/*
Package log provides the ability for microservices to log at different levels
(DEBUG, INFO, WARN, and ERROR). It allows passing of optional fields to the logs.

The package currently makes use of the Zap logger (https://pkg.go.dev/go.uber.org/zap).
This is abstracted so the scope of use is controlled, allowing the underlying technology
to be replaced whenever necessary.
*/
package log
