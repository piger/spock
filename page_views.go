package spock

import (
	"code.google.com/p/xsrftoken"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	ANONYMOUS_NAME  = "Anonymous"
	ANONYMOUS_EMAIL = "anon@wiki.int"
)

type Breadcrumbs struct {
	Pages []string
}

func (b *Breadcrumbs) Add(wikiPath string) {
	if len := len(b.Pages); len > 0 {
		last := b.Pages[len-1]
		if last == wikiPath {
			return
		}
	}

	b.Pages = append(b.Pages, wikiPath)
	numPages := len(b.Pages) - 5
	if numPages < 0 {
		numPages = 0
	}
	b.Pages = b.Pages[numPages:]
}

func init() {
	gob.Register(&Breadcrumbs{})
}

func updateBreadcrumbs(w http.ResponseWriter, r *vRequest, page *Page) []string {
	bcrumbs, ok := r.Session.Values["breadcrumbs"].(*Breadcrumbs)
	if !ok {
		bcrumbs = &Breadcrumbs{}
		r.Session.Values["breadcrumbs"] = bcrumbs
	}
	bcrumbs.Add(page.ShortName())
	r.Session.Save(r.Request, w)

	return bcrumbs.Pages
}

func getBreadcrumbs(r *vRequest) []string {
	if bcrumbs, ok := r.Session.Values["breadcrumbs"].(*Breadcrumbs); ok {
		return bcrumbs.Pages
	}

	return make([]string, 0)
}

func getPagePath(r *vRequest) string {
	vars := mux.Vars(r.Request)
	return vars["pagepath"]
}

// Convert CRLF if the platform is not Windows.
func convertNewlines(text string) string {
	if runtime.GOOS == "windows" {
		return text
	}
	return strings.Replace(text, "\r\n", "\n", -1)
}

func ShowPage(w http.ResponseWriter, r *vRequest) {
	ctx := newTemplateContext(r)

	renderStart := time.Now()

	pagepath := getPagePath(r)
	page, exists, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		EditNewPage(page, w, r)
		return
	}

	html, err := page.Render()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastlog, err := r.Ctx.Storage.GetLastCommit(page.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx["breadcrumbs"] = updateBreadcrumbs(w, r, page)
	ctx["page"] = page
	ctx["content"] = template.HTML(html)
	ctx["lastlog"] = lastlog
	ctx["render_time"] = time.Since(renderStart)
	ctx["alerts"] = GetAlerts(r, w)

	r.Ctx.RenderTemplate("page.html", ctx, w)
}

func EditNewPage(page *Page, w http.ResponseWriter, r *vRequest) {
	ctx := newTemplateContext(r)

	ctx["content"] = template.HTML(NewPageContent)
	ctx["pageName"] = page.ShortName()
	ctx["isNew"] = true
	ctx["comment"] = ""
	ctx["_xsrf"] = xsrftoken.Generate(r.Ctx.XsrfSecret, r.AuthUser.Name, "post")

	r.Ctx.RenderTemplate("edit_page.html", ctx, w)
}

func EditPage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, _, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Request.Method == "POST" {
		if err := r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		xsrf := r.Request.PostFormValue("_xsrf")
		if xsrfValid := xsrftoken.Valid(xsrf, r.Ctx.XsrfSecret, r.AuthUser.Name, "post"); !xsrfValid {
			http.Error(w, "Invalid XSRF token", http.StatusBadRequest)
			return
		}

		content := convertNewlines(r.Request.PostFormValue("content"))
		comment := convertNewlines(r.Request.PostFormValue("comment"))
		doPreview := r.Request.PostFormValue("preview")

		if doPreview != "" {
			ShowPreview(page, content, w, r)
			return
		}

		if comment == "" {
			comment = "(no comment)"
		}
		fullname, email := LookupAuthor(r)

		// Update page RawBytes, header and content with the new data.
		page.RawBytes = []byte(content)
		page.Header, page.Content, err = ParsePageBytes(page.RawBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sig := &CommitSignature{
			Name:  fullname,
			Email: email,
			When:  time.Now(),
		}
		err := r.Ctx.Storage.SavePage(page, sig, comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// index the page
		if err = r.Ctx.Index.AddPage(page); err != nil {
			AddAlert(fmt.Sprintf("bleve: Cannot index document %s: %s\n", page.Path, err), "warning", r)
			r.Session.Save(r.Request, w)
		}

		http.Redirect(w, r.Request, "/"+page.ShortName(), http.StatusSeeOther)
		return
	}

	ctx := newTemplateContext(r)
	ctx["page"] = page
	if len(page.RawBytes) > 0 {
		ctx["content"] = template.HTML(page.RawBytes)
	} else {
		ctx["content"] = template.HTML(NewPageContent)
	}
	ctx["pageName"] = page.ShortName()
	ctx["isNew"] = false
	ctx["comment"] = ""
	ctx["_xsrf"] = xsrftoken.Generate(r.Ctx.XsrfSecret, r.AuthUser.Name, "post")
	ctx["breadcrumbs"] = getBreadcrumbs(r)

	r.Ctx.RenderTemplate("edit_page.html", ctx, w)
}

func LookupAuthor(r *vRequest) (fullname, email string) {
	if ifullname, ok := r.Session.Values["name"]; !ok {
		fullname = ANONYMOUS_NAME
	} else {
		fullname = ifullname.(string)
	}

	if iemail, ok := r.Session.Values["email"]; !ok {
		email = ANONYMOUS_EMAIL
	} else {
		email = iemail.(string)
	}

	return
}

func ShowPreview(page *Page, content string, w http.ResponseWriter, r *vRequest) {
	ctx := newTemplateContext(r)
	ctx["pageName"] = page.ShortName()

	html, err := page.RenderPreview(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx["preview"] = template.HTML(html)
	r.Ctx.RenderTemplate("preview.html", ctx, w)
}

func ShowPageLog(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	commits, err := r.Ctx.Storage.LogsForPage(page.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["pageName"] = page.ShortName()
	ctx["breadcrumbs"] = getBreadcrumbs(r)

	var details []map[string]interface{}

	for _, commitlog := range commits {
		info := make(map[string]interface{})
		info["sha"] = commitlog.Id
		info["message"] = commitlog.Message
		info["name"] = commitlog.Name
		info["email"] = commitlog.Email
		info["when"] = commitlog.When
		details = append(details, info)
	}
	ctx["details"] = details
	r.Ctx.RenderTemplate("log.html", ctx, w)
}

func ListPages(w http.ResponseWriter, r *vRequest) {
	pages, err := r.Ctx.Storage.ListPages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["pages"] = pages
	ctx["breadcrumbs"] = getBreadcrumbs(r)

	r.Ctx.RenderTemplate("ls.html", ctx, w)
}

func RenamePage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	var formError bool
	ctx := newTemplateContext(r)
	ctx["pageName"] = page.ShortName()
	ctx["breadcrumbs"] = getBreadcrumbs(r)
	ctx["_xsrf"] = xsrftoken.Generate(r.Ctx.XsrfSecret, r.AuthUser.Name, "post")

	if r.Request.Method == "POST" {
		if err = r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		xsrf := r.Request.PostFormValue("_xsrf")
		if xsrfValid := xsrftoken.Valid(xsrf, r.Ctx.XsrfSecret, r.AuthUser.Name, "post"); !xsrfValid {
			http.Error(w, "Invalid XSRF token", http.StatusBadRequest)
			return
		}

		newname := r.Request.PostFormValue("new-name")
		comment := convertNewlines(r.Request.PostFormValue("comment"))
		if newname == "" {
			formError = true
		}

		if comment == "" {
			comment = fmt.Sprintf("rename %s to %s", page.Path, newname)
		}

		fullname, email := LookupAuthor(r)
		sig := &CommitSignature{
			Name:  fullname,
			Email: email,
			When:  time.Now(),
		}

		// Rename the page here!
		if !formError {
			_, err = r.Ctx.Storage.RenamePage(page.Path, newname, sig, comment)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r.Request, "/"+newname, http.StatusSeeOther)
		}
	}

	ctx["formError"] = formError

	r.Ctx.RenderTemplate("rename.html", ctx, w)
}

func DiffPage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	vars := mux.Vars(r.Request)
	shaParam := vars["sha"]
	diffs, err := r.Ctx.Storage.DiffPage(page, shaParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["Diffs"] = diffs
	ctx["breadcrumbs"] = getBreadcrumbs(r)
	ctx["pageName"] = page.ShortName()

	r.Ctx.RenderTemplate("diff.html", ctx, w)
}

type SearchResults struct {
	Total   uint64
	Took    time.Duration
	Results []*SearchResult
}

type SearchResult struct {
	Title     string
	Highlight template.HTML
}

func SearchPages(w http.ResponseWriter, r *vRequest) {
	if r.Request.Method != "POST" {
		http.Redirect(w, r.Request, "/index", http.StatusFound)
		return
	}

	if err := r.Request.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	queries, ok := r.Request.Form["q"]
	if !ok {
		log.Println("Empty 'q' parameter for search request")
		http.Redirect(w, r.Request, "/index", http.StatusFound)
		return
	}

	query := queries[0]
	if len(query) < 1 {
		log.Println("Zero length search-query parameter")
		http.Redirect(w, r.Request, "/index", http.StatusFound)
		return
	}

	result, err := r.Ctx.Search(query)
	if err != nil {
		log.Printf("Search error: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	srs := &SearchResults{Total: result.Total, Took: result.Took}
	for _, hit := range result.Hits {
		hl := ""
		for _, fragments := range hit.Fragments {
			// hl += fmt.Sprintf("%s:", fragmentField)
			for _, fragment := range fragments {
				hl += fmt.Sprintf("%s", fragment)
			}
		}
		sr := &SearchResult{Title: hit.ID, Highlight: template.HTML(hl)}
		srs.Results = append(srs.Results, sr)
	}

	ctx := newTemplateContext(r)
	ctx["breadcrumbs"] = getBreadcrumbs(r)
	ctx["SearchQuery"] = query
	ctx["results"] = srs

	r.Ctx.RenderTemplate("results.html", ctx, w)
}

func DeletePage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := r.Ctx.Storage.LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	if r.Request.Method == "POST" {
		if err = r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		xsrf := r.Request.PostFormValue("_xsrf")
		if xsrfValid := xsrftoken.Valid(xsrf, r.Ctx.XsrfSecret, r.AuthUser.Name, "post"); !xsrfValid {
			http.Error(w, "Invalid XSRF token", http.StatusBadRequest)
			return
		}

		comment := convertNewlines(r.Request.PostFormValue("comment"))
		if comment == "" {
			comment = fmt.Sprintf("deleted %s", page.ShortName())
		}

		fullname, email := LookupAuthor(r)
		sig := &CommitSignature{
			Name:  fullname,
			Email: email,
			When:  time.Now(),
		}

		_, err := r.Ctx.Storage.DeletePage(page.Path, sig, comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r.Request, "/index", http.StatusSeeOther)
		return
	}

	ctx := newTemplateContext(r)
	ctx["breadcrumbs"] = getBreadcrumbs(r)
	ctx["_xsrf"] = xsrftoken.Generate(r.Ctx.XsrfSecret, r.AuthUser.Name, "post")
	ctx["pageName"] = page.ShortName()

	r.Ctx.RenderTemplate("delete.html", ctx, w)
}
