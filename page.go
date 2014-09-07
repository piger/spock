// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mschoch/blackfriday-text"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// rst2html program path
var rst2htmlPath string

// NewPageContent is the initial content of a new Wiki page.
var NewPageContent = `---
title: "My page"
description: "A brief page description..."
tags: [ "general" ]
language: "it"
---
# My document title

My first paragraph.
`

const (
	markdownName = "markdown"
	rstName      = "rst"
)

func init() {
	var err error
	if rst2htmlPath, err = lookupRst(); err != nil {
		log.Printf("RestructuredText rendering disabled: %s\n", err)
		rst2htmlPath = "/bin/cat"
	}
}

// PageHeader is the optional YAML header of a wiki page.
type PageHeader struct {
	Title       string
	Description string
	Tags        []string
	Language    string
	Markup      string
}

// Page is a wiki page. The Path attribute contains the relative path
// to the file containing the wiki page (e.g. docs/programming/python.md).
type Page struct {
	Path     string
	Header   *PageHeader
	RawBytes []byte
	Content  []byte
}

// NewPage is the preferred way to create new Page objects.
func NewPage(path string) *Page {
	pageHdr := &PageHeader{}
	page := &Page{Path: path, Header: pageHdr}
	return page
}

func ParsePageBytes(data []byte) (*PageHeader, []byte, error) {
	var content []byte
	ph := &PageHeader{}

	hdrtag := []byte("---")

	// if the first bytes does not contain the YAML header
	if string(data[0:3]) != string(hdrtag) {
		return ph, data, nil
	} else {
		// read and parse the YAML header
		var header []byte

		// find the second yaml marker "---": we skip the first 3 bytes as we need to find
		// the *second* row of "---"; after we have found the position we add back the 3
		// bytes, to account for the first "---". Clear uh?
		mark := bytes.Index(data[len(hdrtag):], hdrtag)
		if mark == -1 {
			return nil, content, errors.New("Cannot find the closing YAML marker")
		}
		mark += len(hdrtag)

		// cross-platform way to find the end of the line
		eolMark := bytes.Index(data[mark:], []byte("\n"))
		if eolMark == -1 {
			return nil, content, errors.New("Cannot find the second newline character")
		}
		headerEnd := mark + eolMark
		header = data[0:headerEnd]
		content = data[headerEnd:]

		err := yaml.Unmarshal(header, ph)
		if err != nil {
			return nil, content, err
		}
	}

	return ph, content, nil
}

// LoadPage loads a page from the filesystem.
func LoadPage(path, relpath string) (*Page, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	page := NewPage(relpath)
	page.RawBytes = data

	page.Header, page.Content, err = ParsePageBytes(data)
	if err != nil {
		return nil, err
	}

	return page, nil
}

// ShortenPageName returns the filename without the extension.
func ShortenPageName(name string) string {
	if ext := filepath.Ext(name); len(ext) > 0 {
		l := len(name) - len(ext)
		return name[0:l]
	}

	return name
}

// ShortName is the "short" (i.e. without the filename extension) name of a page.
func (page *Page) ShortName() string {
	return ShortenPageName(page.Path)
}

// GetMarkup return the page markup based on header informations or filename extension.
func (page *Page) GetMarkup() string {
	if page.Header.Markup != "" {
		return page.Header.Markup
	}
	ext := filepath.Ext(page.Path)
	if ext == ".md" {
		return markdownName
	} else if ext == ".rst" {
		return rstName
	}

	return ""
}

// Render renders the HTML version of a Wiki page.
func (page *Page) Render() ([]byte, error) {
	// if cache, ok := PageCache.Get(page.Path); ok {
	// 	return cache, nil
	// }

	var html []byte
	var err error
	if page.Header.Markup == rstName || strings.HasSuffix(page.Path, ".rst") {
		html, err = renderRst(page.Content)
		// PageCache.Set(page.Path, html)
		return html, err
	} else if page.Header.Markup == markdownName || strings.HasSuffix(page.Path, ".md") || strings.HasSuffix(page.Path, ".txt") {
		html, err = renderMarkdown(page.Content)
		// PageCache.Set(page.Path, html)
		return html, err
	}

	return []byte(page.Content), errors.New("Unknown format")
}

func (page *Page) RenderPlaintext() ([]byte, error) {
	if page.GetMarkup() == markdownName {
		extensions := 0
		renderer := blackfridaytext.TextRenderer()
		return blackfriday.Markdown(page.Content, renderer, extensions), nil
	}

	return page.RawBytes, nil
}

func (page *Page) RenderPreview(content []byte) ([]byte, error) {
	if page.Header.Markup == rstName || strings.HasSuffix(page.Path, ".rst") {
		return renderRst(content)
	} else if page.Header.Markup == markdownName || strings.HasSuffix(page.Path, ".md") || strings.HasSuffix(page.Path, ".txt") {
		return renderMarkdown(content)
	}

	return []byte(page.Content), errors.New("Unknown format")
}

func renderMarkdown(content []byte) ([]byte, error) {
	// Add TOC to the HTML output
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_TOC
	htmlFlags |= blackfriday.HTML_FOOTNOTE_RETURN_LINKS

	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	return blackfriday.Markdown(content, renderer, extensions), nil
}

// Lookup the correct 'rst2html' program inspecting $PATH
func lookupRst() (string, error) {
	var names = []string{"rst2html", "rst2html.py"}

	for _, name := range names {
		if rstbin, err := exec.LookPath(name); err == nil {
			return rstbin, nil
		}
	}

	return "", errors.New("rst2html program not found")
}

func renderRst(content []byte) ([]byte, error) {
	rstTemplate := filepath.Join(DataDir, "rst_template.txt")
	cmd := exec.Command(rst2htmlPath, "--template", rstTemplate)
	cmd.Stdin = strings.NewReader(string(content))
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()

	errStr := string(errout.Bytes())
	if len(errStr) > 0 {
		fmt.Print(errStr)
	}

	return out.Bytes(), err
}
