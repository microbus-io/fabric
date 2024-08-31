/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

package connector

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"

	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/service"
	"gopkg.in/yaml.v3"
)

// SetResFS initialized the connector to load resource files from an arbitrary FS.
func (c *Connector) SetResFS(fs service.FS) error {
	if c.IsStarted() {
		return c.captureInitErr(errors.New("already started"))
	}
	c.resourcesFS = fs
	err := c.initStringBundle()
	if err != nil {
		return c.captureInitErr(errors.Trace(err))
	}
	return nil
}

// SetResFSDir initialized the connector to load resource files from a directory.
func (c *Connector) SetResFSDir(directoryPath string) error {
	err := c.SetResFS(os.DirFS(directoryPath).(service.FS)) // Casting required
	return errors.Trace(err)
}

// ResFS returns the FS associated with the connector.
func (c *Connector) ResFS() service.FS {
	return c.resourcesFS
}

// ReadResFile returns the content of a resource file.
func (c *Connector) ReadResFile(name string) ([]byte, error) {
	b, err := c.resourcesFS.ReadFile(name)
	return b, errors.Trace(err)
}

// MustReadResFile returns the content of a resource file, or nil if not found.
func (c *Connector) MustReadResFile(name string) []byte {
	b, _ := c.resourcesFS.ReadFile(name)
	return b
}

// ReadResTextFile returns the content of a resource file as a string.
func (c *Connector) ReadResTextFile(name string) (string, error) {
	b, err := c.resourcesFS.ReadFile(name)
	return string(b), errors.Trace(err)
}

// MustReadResTextFile returns the content of a resource file as a string, or "" if not found.
func (c *Connector) MustReadResTextFile(name string) string {
	b, _ := c.resourcesFS.ReadFile(name)
	return string(b)
}

// ServeResFile serves the content of a resources file as a response to a web request.
func (c *Connector) ServeResFile(name string, w http.ResponseWriter, r *http.Request) error {
	b, err := c.resourcesFS.ReadFile(name)
	if err != nil {
		return errors.Newc(http.StatusNotFound, "")
	}
	hash := sha256.New()
	hash.Write(b)
	eTag := hex.EncodeToString(hash.Sum(nil))
	w.Header().Set("Etag", eTag)
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

// ExecuteResTemplate parses the resource file as a template, executes it given the data, and returns
// the result. The template is assumed to be a text template unless the file name ends in .html,
// in which case it is processed as an HTML template.
//
// {{ var | attr }}, {{ var | url }}, {{ var | css }} or {{ var | safe }} may be used to prevent the
// escaping of a variable in an HTML template.
// These map to [htmltemplate.HTMLAttr], [htmltemplate.URL], [htmltemplate.CSS] and [htmltemplate.HTML]
// respectively. Use of these types presents a security risk.
func (c *Connector) ExecuteResTemplate(name string, data any) (string, error) {
	b, err := c.resourcesFS.ReadFile(name)
	if err != nil {
		return "", errors.Trace(err)
	}
	var buf bytes.Buffer
	if strings.HasSuffix(strings.ToLower(name), ".html") {
		funcMap := htmltemplate.FuncMap{
			"attr": func(s string) htmltemplate.HTMLAttr {
				return htmltemplate.HTMLAttr(s)
			},
			"url": func(s string) htmltemplate.URL {
				return htmltemplate.URL(s)
			},
			"css": func(s string) htmltemplate.CSS {
				return htmltemplate.CSS(s)
			},
			"safe": func(s string) htmltemplate.HTML {
				return htmltemplate.HTML(s)
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

// initStringBundle reads strings.yaml from the FS into an in-memory map.
func (c *Connector) initStringBundle() error {
	c.stringBundle = nil
	b, err := c.ReadResFile("strings.yaml")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.Trace(err)
	}
	if len(b) == 0 {
		return nil
	}
	var raw map[string]map[string]string
	err = yaml.NewDecoder(bytes.NewReader(b)).Decode(&raw)
	if err != nil {
		return errors.Trace(err)
	}
	if len(raw) == 0 {
		return nil
	}
	// Lowercase the string keys and the language codes
	c.stringBundle = make(map[string]map[string]string, len(raw))
	for k, v := range raw {
		lcKey := strings.ToLower(k)
		c.stringBundle[lcKey] = make(map[string]string, len(v))
		for kk, vv := range v {
			c.stringBundle[lcKey][strings.ToLower(kk)] = vv
		}
	}
	return nil
}

/*
LoadResString returns a string from the string bundle in the language best matched to the locale in the context.
The string bundle is a YAML file that must be loadable from the service's resource FS with the name strings.yaml.
The YAML is expected to be in the following format:

	stringKey:
	  default: Localized
	  en: Localized
	  en-UK: Localised
	  fr: Localisée

If a default is not provided, English (en) is used as the fallback language.
String keys and locale names are case insensitive.
*/
func (c *Connector) LoadResString(ctx context.Context, stringKey string) (string, error) {
	if c.stringBundle == nil {
		return "", errors.New("string bundle strings.yaml is not found in resource FS")
	}
	txl := c.stringBundle[strings.ToLower(stringKey)]
	if txl == nil {
		return "", errors.Newf("no string matches the key '%s'", stringKey)
	}
	// da, en-gb;q=0.8, en;q=0.7
	full := frame.Of(ctx).Header().Get("Accept-Language")
	var qMax float64
	segments := strings.Split(full, ",")
	var result string
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		var q float64
		lang, after, found := strings.Cut(seg, ";")
		if !found {
			// da
			q = 1.0
		} else {
			// en-gb;q=0.8
			qStr := strings.TrimLeft(after, " q=")
			q, _ = strconv.ParseFloat(qStr, 64)
		}
		lang = strings.ToLower(lang)
		if q <= qMax {
			continue
		}
		for {
			v, ok := txl[lang]
			if ok {
				result = v
				qMax = q
				break
			}
			lang, _, found = strings.Cut(lang, "-")
			if !found {
				break
			}
		}
	}
	if qMax == 0 && result == "" {
		var ok bool
		result, ok = txl["default"]
		if !ok {
			result = txl["en"]
		}
	}
	return result, nil
}
