/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package connector

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"gopkg.in/yaml.v3"
)

type FS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

// SetResFS initialized the connector to load resource files from an arbitrary FS.
func (c *Connector) SetResFS(resFS FS) error {
	if c.started {
		return c.captureInitErr(errors.New("already started"))
	}
	c.resourcesFS = resFS
	err := c.initStringBundle()
	if err != nil {
		return c.captureInitErr(err)
	}
	return nil
}

// initStringBundle reads strings.yaml from the FS into an in-memory map.
func (c *Connector) initStringBundle() error {
	b, err := c.ReadResFile("strings.yaml")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.Trace(err)
	}
	if len(b) > 0 {
		err = yaml.NewDecoder(bytes.NewReader(b)).Decode(&c.stringBundle)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// SetResDirFS initialized the connector to load resource files from a directory.
func (c *Connector) SetResDirFS(name string) error {
	err := c.SetResFS(os.DirFS(name).(FS)) // Casting required
	return errors.Trace(err)
}

// ReadResDir returns the entries in the resource directory.
func (c *Connector) ReadResDir(name string) ([]fs.DirEntry, error) {
	entries, err := c.resourcesFS.ReadDir(name)
	return entries, errors.Trace(err)
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

// LoadResString returns a string from the string bundle in the language best matched to the context.
// The string bundle must be loadable from the service's FS with the name strings.yaml.
func (c *Connector) LoadResString(ctx context.Context, key string) string {
	if c.stringBundle == nil {
		return ""
	}
	str := c.stringBundle[key]
	if str == nil {
		return ""
	}
	languages := frame.Of(ctx).Languages()
	for _, language := range languages {
		for {
			val := str[language]
			if val != "" {
				return val
			}
			p := strings.LastIndex(language, "-")
			if p < 0 {
				break
			}
			language = language[:p]
		}
	}
	return str["en"] // Default to English
}
