/*
Copyright (c) 2023-2025 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package mem provides pooled byte buffers to reduce garbage-collector pressure on hot paths. Buffers are drawn from a
fixed set of power-of-two size classes, from 1KB up to 4MB.

Alloc returns a zero-length buffer with capacity for at least the requested size, drawn from the smallest size class
that fits; a request larger than the largest class is served by a plain allocation of that capacity. Free returns a
buffer to its size class for reuse, but
recycles only a buffer whose capacity exactly matches a class - one not obtained from Alloc, grown past its class by
append, or larger than the maximum class is discarded. Copy allocates a buffer via Alloc and copies the given bytes
into it.

Because the size classes are fungible, a request is rounded up to the next class, so a 4.1KB request occupies an 8KB
buffer. The pool is therefore intended for short-lived buffers that are allocated, used, and freed quickly, not for
long-lived retention where the rounding would waste memory for the whole hold. After Free, the buffer must not be read
or written, as a later Alloc may hand its bytes to another caller.
*/
package mem
