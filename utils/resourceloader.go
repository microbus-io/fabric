/*
Copyright 2023 Microbus LLC and various contributors

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

package utils

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	htmltemplate "html/template"
	"net/http"
	"strconv"
	"strings"
	texttemplate "text/template"

	"github.com/microbus-io/fabric/errors"
)

// ResourceLoader extends an embedded FS with convenience methods.
type ResourceLoader struct {
	embed.FS
}

// LoadFile returns the content of the embedded file, or nil if not found.
func (rl ResourceLoader) LoadFile(name string) []byte {
	b, _ := rl.ReadFile(name)
	return b
}

// LoadText returns the content of the embedded file as a string, or nil if not found.
func (rl ResourceLoader) LoadText(name string) string {
	b, _ := rl.ReadFile(name)
	return string(b)
}

// ServeFile serves the content of the embedded file as a response of a web request.
func (rl ResourceLoader) ServeFile(name string, w http.ResponseWriter, r *http.Request) error {
	b, err := rl.ReadFile(name)
	if err != nil {
		return errors.Newc(http.StatusNotFound, "")
	}
	hash := sha256.New()
	hash.Write(b)
	eTag := hex.EncodeToString(hash.Sum(nil))
	w.Header().Set("ETag", eTag)
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" {
		w.Header().Set("Cache-Control", "max-age=3600, private, stale-while-revalidate=3600")
	}
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(b)
		w.Header().Set("Content-Type", contentType)
	}
	if r.Header.Get("If-None-Match") == eTag {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, err = w.Write(b)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// LoadTemplate parses the embedded file as a template, executes it given the data, and returns
// the result. The template is assumed to be a text template unless the file name ends in .html.
func (rl ResourceLoader) LoadTemplate(name string, data any) (string, error) {
	b, err := rl.ReadFile(name)
	if err != nil {
		return "", errors.Trace(err)
	}
	var buf bytes.Buffer
	if strings.HasSuffix(strings.ToLower(name), ".html") {
		funcMap := htmltemplate.FuncMap{
			"attr": func(s string) htmltemplate.HTMLAttr {
				return htmltemplate.HTMLAttr(s)
			},
			"safe": func(s string) htmltemplate.HTML {
				return htmltemplate.HTML(s)
			},
			"url": func(s string) htmltemplate.URL {
				return htmltemplate.URL(s)
			},
			"css": func(s string) htmltemplate.CSS {
				return htmltemplate.CSS(s)
			},
		}
		htmlTmpl, err := htmltemplate.New(name).Funcs(funcMap).Parse(string(b))
		if err != nil {
			return "", errors.Trace(err)
		}
		err = htmlTmpl.ExecuteTemplate(&buf, name, data)
		if err != nil {
			return "", errors.Trace(err)
		}
	} else {
		textTmpl, err := texttemplate.New(name).Parse(string(b))
		if err != nil {
			return "", errors.Trace(err)
		}
		err = textTmpl.ExecuteTemplate(&buf, name, data)
		if err != nil {
			return "", errors.Trace(err)
		}
	}
	return buf.String(), nil
}
