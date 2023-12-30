# Package `clock`

The `clock` package is an abstraction of the functions in the standard library `time` package. It includes two implementations: a real clock and a mock clock. The first
is a real-time clock which simply wraps the `time` package's functions. The
second is a mock clock that changes only change
programmatically adjusted and is ideal for testing time-sensitive functions.

A mock clock can be assigned to a `Connector` in the `LOCAL` or `TESTINGAPP` deployment environments. The mock clock is for the application developer to test time-sensitive code. It plays no part in any of the framework functions, such as ticker execution schedule or timeout management.

For example:

```go
mockClock := clock.NewMock()
con.SetClock(mockClock)
```
