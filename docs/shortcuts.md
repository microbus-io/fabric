# Shortcuts

* The timeouts for the `OnStartup` and `OnShutdown` callbacks are hard-coded to `time.Minute`
* The network hop duration is hard-coded to `250 * time.Millisecond`
* The logger is rudimentary
* The known responders hashmap is unbound and theoretically might overflow
