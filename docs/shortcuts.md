# Shortcuts

* The timeouts for the `OnStartup` and `OnShutdown` callbacks are hard-coded to `time.Minute`
* The network hop duration is hard-coded to `250 * time.Millisecond`
* The known responders hashmap is unbounded and theoretically might overflow
* There's an 8-second hard-coded timeout for all fragments of a large message to be reassembled
