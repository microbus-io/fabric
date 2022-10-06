/*
Package log provides the ability to create log fields. These fields are passed along in the logs.
The package currently makes use of the Zap logger (https://pkg.go.dev/go.uber.org/zap), although
abstracts this away so the scope of use is controlled. Abstracting this also allows the underlying technology
to be replaced whenever necessary.

The connector implements the loggers so that microservices can log at DEBUG, INFO, WARN, and ERROR levels,
and optionally pass along these log fields.
*/
package log
