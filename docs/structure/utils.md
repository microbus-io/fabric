# Package `utils`

Package `utils` includes various independent utilities.

`SourceCodeSHA256` reads the content of a source code directory and generates a SHA256 of its relevant content. It is used by the code generator for change detection and automatic versioning.

`CatchPanic` is a utility function that [converts panics into standard errors](../tech/errorcapture.md). It is used by the `Connector` to wrap callbacks to user code.

`InfiniteChan` is a channel backed by a finite channel and an infinite queue. Elements that cannot be written to the channel are instead pushed to the queue and are delivered to the channel when capacity frees up. `InfiniteChan` therefore never blocks on write but may block on read. Queued elements may be dropped if the channel is closed and left unread for over the idle timeout.
