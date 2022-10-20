# Package `clock`

The `clock` package is an abstraction of the functions in the standard library `time` package. It includes two implementations: a real clock and a mock clock. The first
is a real-time clock which simply wraps the `time` package's functions. The
second is a mock clock which will only change when
programmatically adjusted and is ideal for testing time-sensitive functions.
