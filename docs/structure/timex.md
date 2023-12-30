# Package `timex`

Package `timex` enhances the standard `time.Time` to make it more compatible with JavaScript and SQL. It adjusts JSON serialization of the zero time to `null` and infers the layout when parsing a string. It also implements the `Scanner` and `Valuer` interfaces for SQL integration.
