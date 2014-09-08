// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"github.com/blevesearch/bleve"
)

func (ac *AppContext) Search(searchQuery string) (*bleve.SearchResult, error) {
	// query := bleve.NewQueryStringQuery(searchQuery)
	queryEn := bleve.NewMatchQuery(searchQuery).SetField("body_en")
	queryIt := bleve.NewMatchQuery(searchQuery).SetField("body_it")
	queryTitle := bleve.NewMatchQuery(searchQuery).SetField("title")
	query := bleve.NewDisjunctionQuery([]bleve.Query{queryEn, queryIt, queryTitle})
	req := bleve.NewSearchRequest(query)
	req.Highlight = bleve.NewHighlight()
	return ac.Index.index.Search(req)
}
