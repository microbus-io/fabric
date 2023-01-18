/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

package httpingress

import "net/http"

// PassThrough wraps calls to ResponseWriter, collecting metrics in the process.
type PassThrough struct {
	W  http.ResponseWriter
	N  int
	SC int
}

func (pt *PassThrough) Header() http.Header {
	return pt.W.Header()
}

func (pt *PassThrough) Write(b []byte) (int, error) {
	n, err := pt.W.Write(b)
	pt.N += n
	return n, err // No trace
}

func (pt *PassThrough) WriteHeader(statusCode int) {
	pt.SC = statusCode
	pt.W.WriteHeader(statusCode)
}
