# Package `utils`

Package `utils` includes various independent utilities.

`SourceCodeSHA256` reads the content of a source code directory and generates a SHA256 of its relevant content. It is used by the code generator for change detection and automatic versioning.

`CatchPanic` is a utility function that [converts panics into standard errors](../blocks/error-capture.md). It is used by the `Connector` to wrap callbacks to user code.

`SyncMap` is a thin wrapper over a subset of the operations of the standard `sync.Map`. It introduces generics to make these more type-safe.
