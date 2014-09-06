// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"github.com/blevesearch/bleve"
	"log"
)

const textAnalyzer = "standard"
const textEnAnalyzer = "en"
const textItAnalyzer = "it"

type WikiPage struct {
	Title  string `json:"title"`
	BodyEn string `json:"body_en"`
	BodyIt string `json:"body_it"`
}

func (wp *WikiPage) Type() string {
	return "wikiPage"
}

type Index struct {
	index bleve.Index
	path  string
}

func buildIndexMapping() *bleve.IndexMapping {

	enTextMapping := bleve.NewTextFieldMapping()
	enTextMapping.Analyzer = textEnAnalyzer

	itTextMapping := bleve.NewTextFieldMapping()
	// XXX the Italian analyzer is giving wrong results.
	// itTextMapping.Analyzer = textItAnalyzer
	itTextMapping.Analyzer = textAnalyzer

	stdTextMapping := bleve.NewTextFieldMapping()
	stdTextMapping.Analyzer = textAnalyzer

	wikiPageMapping := bleve.NewDocumentMapping()
	wikiPageMapping.AddFieldMappingsAt("title", stdTextMapping)
	wikiPageMapping.AddSubDocumentMapping("id", bleve.NewDocumentDisabledMapping())
	wikiPageMapping.AddFieldMappingsAt("body_en", enTextMapping)
	wikiPageMapping.AddFieldMappingsAt("body_it", itTextMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("wikiPage", wikiPageMapping)

	return mapping
}

func OpenIndex(path string) (*Index, error) {
	index, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Println("Creating a new search index")
		indexMapping := buildIndexMapping()
		index, err = bleve.New(path, indexMapping)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &Index{index: index, path: path}, nil
}

func (idx *Index) AddPage(page *Page) error {
	wikiPage, err := page.ToWikiPage()
	if err != nil {
		return err
	}
	return idx.index.Index(page.ShortName(), wikiPage)
}

func (idx *Index) DeletePage(page *Page) error {
	return idx.index.Delete(page.ShortName())
}

func (idx *Index) Close() {
	idx.index.Close()
}

func (idx *Index) DocCount() uint64 {
	return idx.index.DocCount()
}

func (idx *Index) IndexWiki(storage Storage) error {
	pages, err := storage.ListPages()
	if err != nil {
		log.Printf("Cannot get page list: %s\n", err)
		return err
	} else if len(pages) == 0 {
		return nil
	}

	batch := bleve.NewBatch()
	for _, pagePath := range pages {
		page, _, err := storage.LookupPage(pagePath)
		if err != nil {
			log.Printf("Error loading page %s: %s\n", pagePath, err)
			continue
		}

		wikiPage, err := page.ToWikiPage()
		if err != nil {
			log.Printf("Error converting page %s for indexing: %s\n", page.ShortName(), err)
			continue
		}

		batch.Index(page.ShortName(), wikiPage)
	}

	err = idx.index.Batch(batch)
	if err != nil {
		log.Printf("Error executing index batch: %s\n", err)
	}

	return err
}

func (page *Page) ToWikiPage() (*WikiPage, error) {
	text, err := page.RenderPlaintext()
	if err != nil {
		return nil, err
	}
	body := string(text)

	if page.Header.Language == "it" {
		return &WikiPage{Title: page.ShortName(), BodyIt: body}, nil
	}
	return &WikiPage{Title: page.ShortName(), BodyEn: body}, nil
}
