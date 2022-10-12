/*
Package frag implements means to break large HTTP requests and responses into fragments
that can then be reassembled.
Fragmentation is required because NATS sets a limit to the size for messages
that can be transferred on the bus
*/
package frag
