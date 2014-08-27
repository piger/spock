package spock

import (
	"bytes"
	"errors"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"log"
)

// the optional YAML header of a wiki page.
type PageHeader struct {
	Title       string
	Description string
	Tags        []string
	Language    string
	Markup      string
}

// A wiki page. The Path attribute contains the relative path to the file
// containing the wiki page (e.g. docs/programming/python.md).
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

		// find the second yaml marker "---": we skip the first 3 bytes as we need to find
		// the *second* row of "---"; after we have found the position we add back the 3
		// bytes, to account for the first "---". Clear uh?
		mark := bytes.Index(data[3:], []byte("---"))
		if mark == -1 {
			return nil, errors.New("Cannot find the closing YAML marker")
		}
		mark += 3

		// cross-platform way to find the end of the line
		eolMark := bytes.Index(data[mark:], []byte("\n"))
		if eolMark == -1 {
			return nil, errors.New("Cannot find the second newline character")
		}
		// YAML header ends at mark, but mark is a relative position; to fix this
		// we add the length of the first "---" and the length of the second "---"
		// plus a newline character.
		headerEnd := mark + eolMark
		header = data[0:headerEnd]
		page.Content = data[headerEnd:]

		err = yaml.Unmarshal(header, &page.Header)
		if err != nil {
			return nil, err
		}
	}

	return page, nil
}

func ShortenPageName(name string) string {
	if ext := filepath.Ext(name); len(ext) > 0 {
		l := len(name) - len(ext)
		return name[0:l]
	} else {
		return name
	}
}

// A "short name" for a wiki page complete path.
func (page *Page) ShortName() string {
	ext := filepath.Ext(page.Path)
	if len(ext) > 0 {
		l := len(page.Path) - len(ext)
		return page.Path[0:l]
	} else {
		return page.Path
	}
}

func (page *Page) Render() ([]byte, error) {
	if page.Header.Markup == "rst" || strings.HasSuffix(page.Path, ".rst") {
		return renderRst(page.Content)
	} else if page.Header.Markup == "markdown" || strings.HasSuffix(page.Path, ".md") || strings.HasSuffix(page.Path, ".txt") {
		return renderMarkdown(page.Content)
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
	rst2html, err := exec.LookPath("rst2html")
	if err != nil {
		return []byte(""), err
	}

	cmd := exec.Command(rst2html, "--template", "data/rst_template.txt")
	cmd.Stdin = strings.NewReader(string(content))
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err = cmd.Run()

	errStr := string(errout.Bytes())
	if len(errStr) > 0 {
		log.Print(string(errout.Bytes()))
	}

	return out.Bytes(), err
}
