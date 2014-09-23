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
	"time"
)

// rst2html program path
var rst2htmlPath string

// NewPageContent is the initial content of a new Wiki page.
var NewPageContent = `---
title: "My page"
description: "A brief page description..."
language: "en"
---
# My document title

My first paragraph.
`

const (
	markdownName = "markdown"
	rstName      = "rst"

	DefaultExtension = "md"
)

var (
	htmlBodyStart = []byte("<body>")
	htmlBodyEnd   = []byte("</body>")
	headerTag     = []byte("---")
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
	Mtime    time.Time
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

	// if the first bytes does not contain the YAML header
	if !bytes.Equal(data[0:len(headerTag)], headerTag) {
		return ph, data, nil
	} else {
		// read and parse the YAML header
		var header []byte

		// find the second yaml marker "---": we skip the first 3 bytes as we need to find
		// the *second* row of "---"; after we have found the position we add back the 3
		// bytes, to account for the first "---". Clear uh?
		mark := bytes.Index(data[len(headerTag):], headerTag)
		if mark == -1 {
			return nil, content, errors.New("Cannot find the closing YAML marker")
		}
		mark += len(headerTag)

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

// LoadPage loads a page from the filesystem; the "path" argument must be an
// absolute filename, and the "relpath" must be relative "wiki path" plus
// the file extension; example arguments:
// "/var/spock/repository/notes/Linux.md" and "notes/Linux.md".
func LoadPage(path, relpath string) (*Page, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("page path %s is not an absolute path", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	page := NewPage(relpath)
	if err := page.SetRawBytes(data); err != nil {
		return nil, err
	}
	page.Mtime = fi.ModTime()

	return page, nil
}

func (page *Page) SetRawBytes(content []byte) (err error) {
	page.RawBytes = content
	page.Header, page.Content, err = ParsePageBytes(page.RawBytes)
	if err == nil {
		page.Mtime = time.Now()
	}
	return err
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

func (page *Page) String() string {
	return fmt.Sprintf("Page[%s]", page.Path)
}

// GetMarkup return the page markup based on header informations or filename extension.
func (page *Page) GetMarkup() (markup string) {
	if page.Header.Markup != "" {
		markup = page.Header.Markup
	} else {
		ext := filepath.Ext(page.Path)
		switch ext {
		case ".md", ".txt":
			markup = markdownName
		case ".rst":
			markup = rstName
		default:
			markup = markdownName // XXX default
		}
	}
	return
}

// Render renders the HTML version of a Wiki page.
func (page *Page) Render() (html []byte, err error) {
	markup := page.GetMarkup()
	switch markup {
	case markdownName:
		html, err = renderMarkdown(page.Content)
	case rstName:
		html, err = renderRst(page.Content)
	default:
		html, err = []byte(page.Content), fmt.Errorf("Unknown format: %s", markup)
	}
	return html, err
}

func (page *Page) RenderPlaintext() (txt []byte, err error) {
	switch page.GetMarkup() {
	case markdownName:
		extensions := 0
		renderer := blackfridaytext.TextRenderer()
		txt, err = blackfriday.Markdown(page.Content, renderer, extensions), nil
	default:
		// we won't return an error because text rendering is "best effort" :)
		txt, err = page.RawBytes, nil
	}
	return txt, err
}

func (page *Page) RenderPreview(content []byte) (html []byte, err error) {
	markup := page.GetMarkup()
	switch markup {
	case markdownName:
		html, err = renderMarkdown(content)
	case rstName:
		html, err = renderRst(content)
	default:
		html, err = []byte(content), fmt.Errorf("Unknown format: %s", markup)
	}
	return html, err
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
	cmd := exec.Command(rst2htmlPath)
	cmd.Stdin = bytes.NewReader(content)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()

	errStr := errout.String()
	if len(errStr) > 0 {
		fmt.Print(errStr)
	}

	html := out.Bytes()
	bs := bytes.Index(html, htmlBodyStart)
	if bs == -1 {
		return nil, fmt.Errorf("Error rendering rst: cannot find <body> tag")
	}
	be := bytes.Index(html, htmlBodyEnd)
	if be == -1 {
		return nil, fmt.Errorf("Error rendering rst: cannot find </body> tag")
	}

	html = html[bs+len(htmlBodyStart) : be]
	return html, err
}
