# Package `utils`

Package `utils` includes various independent utilities.

`SourceCodeSHA256` reads the content of a source code directory and generates a SHA256 of its relevant content. It is used by the code generator for change detection and automatic versioning.

`SyncMap` is a thin wrapper over a subset of the operations of the standard `sync.Map`. It introduces generics to make these more type-safe.
