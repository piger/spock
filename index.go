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
