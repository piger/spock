// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"github.com/blevesearch/bleve"
	bleveDocument "github.com/blevesearch/bleve/document"
	"log"
	"path/filepath"
	"time"
)

const (
	// IndexDirName is the name of the directory containing the bleve index
	IndexDirName = ".bleve"

	textAnalyzer   = "standard"
	textEnAnalyzer = "en"
	textItAnalyzer = "it"
)

type WikiPage struct {
	Title  string    `json:"title"`
	BodyEn string    `json:"body_en"`
	BodyIt string    `json:"body_it"`
	Body   string    `json:"body"`
	Mtime  time.Time `json:"mtime"`
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
	itTextMapping.Analyzer = textItAnalyzer

	stdTextMapping := bleve.NewTextFieldMapping()
	stdTextMapping.Analyzer = textAnalyzer

	dtMapping := bleve.NewDateTimeFieldMapping()

	wikiPageMapping := bleve.NewDocumentMapping()
	wikiPageMapping.AddFieldMappingsAt("title", stdTextMapping)
	wikiPageMapping.AddSubDocumentMapping("id", bleve.NewDocumentDisabledMapping())
	wikiPageMapping.AddFieldMappingsAt("body_en", enTextMapping)
	wikiPageMapping.AddFieldMappingsAt("body_it", itTextMapping)
	wikiPageMapping.AddFieldMappingsAt("body", stdTextMapping)
	wikiPageMapping.AddFieldMappingsAt("mtime", dtMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("wikiPage", wikiPageMapping)

	mapping.DefaultAnalyzer = textAnalyzer

	return mapping
}

func OpenIndex(basepath string) (*Index, error) {
	path := filepath.Join(basepath, IndexDirName)

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

func (idx *Index) DocCount() (uint64, error) {
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

	batch := idx.index.NewBatch()
	for _, pagePath := range pages {
		page, _, err := storage.LookupPage(pagePath)
		if err != nil {
			log.Printf("Error loading page %s: %s\n", pagePath, err)
			continue
		}

		// try to skip already indexed documents by checking the mtime
		doc, err := idx.index.Document(page.ShortName())
		if err != nil {
			log.Printf("error getting document %s from index: %s\n", page.ShortName(), err)
		} else if doc != nil {
			for _, field := range doc.Fields {
				if field.Name() != "mtime" {
					continue
				}
				if dtfield, ok := field.(*bleveDocument.DateTimeField); ok {
					if dt, err := dtfield.DateTime(); err == nil {
						if dt.Equal(page.Mtime) || dt.After(page.Mtime) {
							continue
						} else {
							log.Printf("Reindexing \"%s\"\n", page.ShortName())
						}
					}
				}
			}
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
	wp := &WikiPage{Title: page.ShortName(), Mtime: page.Mtime}

	if page.Header.Language == "it" {
		wp.BodyIt = body
	} else if page.Header.Language == "en" {
		wp.BodyEn = body
	} else {
		wp.Body = body
	}
	return wp, nil
}

func (ac *AppContext) Search(searchQuery string, size, from int) (*bleve.SearchResult, error) {
	// query := bleve.NewQueryStringQuery(searchQuery)
	queryEn := bleve.NewMatchQuery(searchQuery).SetField("body_en")
	queryIt := bleve.NewMatchQuery(searchQuery).SetField("body_it")
	queryStd := bleve.NewMatchQuery(searchQuery).SetField("body")
	queryTitle := bleve.NewMatchQuery(searchQuery).SetField("title")
	query := bleve.NewDisjunctionQuery([]bleve.Query{queryEn, queryIt, queryStd, queryTitle})

	req := bleve.NewSearchRequestOptions(query, 100, 0, false)
	req.Highlight = bleve.NewHighlight()
	return ac.Index.index.Search(req)
}
