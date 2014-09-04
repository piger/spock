package spock

import (
	"github.com/blevesearch/bleve"
)

func (ac *AppContext) Search(searchQuery string) (*bleve.SearchResult, error) {
	query := bleve.NewMatchQuery(searchQuery)
	req := bleve.NewSearchRequest(query)
	req.Highlight = bleve.NewHighlight()
	return ac.Index.index.Search(req)
}
