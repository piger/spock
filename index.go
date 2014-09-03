package spock

import (
	"github.com/blevesearch/bleve"
	"log"
)

const textAnalyzer = "simple"
const textEnAnalyzer = "en"
const textItAnalyzer = "it"

type WikiPage struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	BodyEn string `json:"body_en"`
	BodyIt string `json:"body_it"`
}

type Index struct {
	index bleve.Index
	path  string
}

func buildIndexMapping() *bleve.IndexMapping {
	titleMapping := bleve.NewDocumentMapping().
		AddFieldMapping(
		bleve.NewFieldMapping("", "text", textAnalyzer, true, true, true, true))

	bodyEnMapping := bleve.NewDocumentMapping().
		AddFieldMapping(
		bleve.NewFieldMapping("", "text", textEnAnalyzer, true, true, true, true))

	bodyItMapping := bleve.NewDocumentMapping().
		AddFieldMapping(
		bleve.NewFieldMapping("", "text", textItAnalyzer, true, true, true, true))

	wikiPageMapping := bleve.NewDocumentMapping().
		AddSubDocumentMapping("title", titleMapping).
		AddSubDocumentMapping("body_en", bodyEnMapping).
		AddSubDocumentMapping("body_it", bodyItMapping)

	indexMapping := bleve.NewIndexMapping().AddDocumentMapping("page", wikiPageMapping)
	indexMapping.DefaultAnalyzer = textEnAnalyzer
	return indexMapping
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

func (idx *Index) AddPage(path string, wikiPage *WikiPage) error {
	return idx.index.Index(path, wikiPage)
}

func (idx *Index) Close() {
	idx.index.Close()
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
