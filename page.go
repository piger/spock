package spock

import (
	"bytes"
	"errors"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type PageHeader struct {
	Title       string
	Description string
	Tags        []string
	Language    string
	Markup      string
}

type Page struct {
	Path     string
	Header   *PageHeader
	RawBytes []byte
	Content  []byte
}

func LoadPage(path, relpath string) (*Page, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	page := &Page{Path: relpath, RawBytes: data}

	if string(data[0:3]) != "---" {
		page.Content = data
	} else {
		var header []byte

		// find the second yaml marker "---"
		mark := bytes.Index(data[3:], []byte("---"))
		if mark == -1 {
			return nil, errors.New("Cannot find the closing YAML marker")
		}
		// YAML header ends at mark, but mark is a relative position; to fix this
		// we add the length of the first "---" and the length of the second "---"
		// plus a newline character.
		headerEnd := mark + len("---") + len("---\n")
		header = data[0:headerEnd]
		page.Content = data[headerEnd+1:]

		err = yaml.Unmarshal(header, &page.Header)
		if err != nil {
			return nil, err
		}
	}

	return page, nil
}

func (page *Page) Render() ([]byte, error) {
	if strings.HasSuffix(page.Path, ".md") || strings.HasSuffix(page.Path, ".txt") {
		return renderMarkdown(page.Content)
	} else if strings.HasSuffix(page.Path, ".rst") {
		return renderRst(page.Content)
	}

	return []byte(page.Content), errors.New("Unknown format")
}

func renderMarkdown(content []byte) ([]byte, error) {
	// Add TOC to the HTML output
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_TOC

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

func renderRst(content []byte) ([]byte, error) {
	rst2html, err := exec.LookPath("rst2html.py")
	if err != nil {
		return []byte(""), nil
	}

	cmd := exec.Command(rst2html, "--template", "data/rst_template.txt")
	cmd.Stdin = strings.NewReader(string(content))
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	return out.Bytes(), err
}