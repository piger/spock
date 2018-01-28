// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"bytes"
	"go.marzhillstudios.com/pkg/go-html-transform/h5"
	"go.marzhillstudios.com/pkg/go-html-transform/html/transform"
	"golang.org/x/net/html"
	"path"
	"strings"
)

const (
	NewPageCSSClass = "new-page"
)

func getAttribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

type LinkCollector struct {
	Pages []string
	Path  string
}

func (lc LinkCollector) Find(rootNode *html.Node) []*html.Node {
	var result []*html.Node

	pagemap := make(map[string]bool)
	for _, page := range lc.Pages {
		pagemap["/"+page] = true
	}

	tree := h5.NewTree(rootNode)
	tree.Walk(func(node *html.Node) {
		if node.Data != "a" {
			return
		}
		href := getAttribute(node, "href")

		// exclude external URLs
		if strings.Index(href, "://") != -1 {
			return
		}

		if href[0] != '/' {
			href = path.Join(lc.Path, href)
		}

		if _, ok := pagemap[href]; !ok {
			result = append(result, node)
		}
	})

	return result
}

func AddNewPageClass(node *html.Node) {
	var attr *html.Attribute
	for i, a := range node.Attr {
		if a.Key == "class" {
			attr = &node.Attr[i]
			break
		}
	}
	if attr == nil {
		attr = &html.Attribute{Key: "class", Val: NewPageCSSClass}
		node.Attr = append(node.Attr, *attr)
	} else {
		attr.Val += " " + NewPageCSSClass
	}
}

// AddCSSClasses add the "new-page" CSS class to all the non-existent pages
// linked in a HTML page. The arguments are: a list of wiki pages, the dir
// component of the current page (used for relative links) and the raw HTML.
func AddCSSClasses(pages []string, path string, html []byte) ([]byte, error) {
	rdr := bytes.NewReader(html)
	tree, err := h5.New(rdr)
	if err != nil {
		return nil, err
	}
	trans := transform.New(tree)
	tf := &LinkCollector{pages, path}
	trans.ApplyWithCollector(AddNewPageClass, tf)

	wrt := &bytes.Buffer{}
	err = trans.Render(wrt)
	if err != nil {
		return nil, err
	}
	return wrt.Bytes(), nil
}
