package spock

import (
	"code.google.com/p/xsrftoken"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"time"
)

const (
	ANONYMOUS_NAME  = "Anonymous"
	ANONYMOUS_EMAIL = "anon@wiki.int"
)

func getPagePath(r *vRequest) string {
	vars := mux.Vars(r.Request)
	return vars["pagepath"]
}

func ShowPage(w http.ResponseWriter, r *vRequest) {
	ctx := newTemplateContext(r)

	renderStart := time.Now()

	pagepath := getPagePath(r)
	page, exists, err := (*r.Ctx.Storage).LookupPage(pagepath)
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

	lastlog, err := (*r.Ctx.Storage).GetLastCommit(page.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx["pageName"] = page.ShortName()
	ctx["content"] = template.HTML(html)
	ctx["author_email"] = lastlog.Email
	ctx["author_name"] = lastlog.Name
	ctx["modification_time"] = lastlog.When
	ctx["last_commit"] = lastlog.Message
	ctx["render_time"] = time.Since(renderStart)

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
	page, _, err := (*r.Ctx.Storage).LookupPage(pagepath)
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

		content := r.Request.PostFormValue("content")
		comment := r.Request.PostFormValue("comment")
		doPreview := r.Request.PostFormValue("preview")

		if doPreview != "" {
			ShowPreview(page, content, w, r)
			return
		}

		if comment == "" {
			comment = "(no comment)"
		}
		fullname, email := LookupAuthor(r)

		page.RawBytes = []byte(content)
		sig := &CommitSignature{
			Name:  fullname,
			Email: email,
			When:  time.Now(),
		}
		err := (*r.Ctx.Storage).SavePage(page, sig, comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r.Request, "/"+page.ShortName(), http.StatusSeeOther)
		return
	}

	ctx := newTemplateContext(r)
	ctx["page"] = page
	ctx["content"] = template.HTML(page.RawBytes)
	ctx["pageName"] = page.ShortName()
	ctx["isNew"] = false
	ctx["comment"] = ""
	ctx["_xsrf"] = xsrftoken.Generate(r.Ctx.XsrfSecret, r.AuthUser.Name, "post")

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

	html, err := page.RenderContent(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx["preview"] = template.HTML(html)
	r.Ctx.RenderTemplate("preview.html", ctx, w)
}

func ShowPageLog(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := (*r.Ctx.Storage).LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	commits, err := (*r.Ctx.Storage).LogsForPage(page.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["pageName"] = page.ShortName()
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
	pages, err := (*r.Ctx.Storage).ListPages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["pages"] = pages

	r.Ctx.RenderTemplate("ls.html", ctx, w)
}

func RenamePage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, exists, err := (*r.Ctx.Storage).LookupPage(pagepath)
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

	if r.Request.Method == "POST" {
		if err = r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		newname := r.Request.PostFormValue("new-name")
		comment := r.Request.PostFormValue("comment")
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
			_, _, err = (*r.Ctx.Storage).RenamePage(page.Path, newname, sig, comment)
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
	page, exists, err := (*r.Ctx.Storage).LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !exists {
		http.NotFound(w, r.Request)
		return
	}

	vars := mux.Vars(r.Request)
	shaParam := vars["sha"]
	diffs, err := (*r.Ctx.Storage).DiffPage(page, shaParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := newTemplateContext(r)
	ctx["Diffs"] = diffs

	r.Ctx.RenderTemplate("diff.html", ctx, w)
}

type searchResult struct {
	Title     string
	Lang      string
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

	ctx := newTemplateContext(r)
	ctx["SearchQuery"] = query
	ctx["Suggestion"] = result.Suggestion

	var rv []*searchResult
	for _, r := range result.Results {
		sr := &searchResult{Title: ShortenPageName(r.Title), Lang: r.Lang, Highlight: template.HTML(r.Highlight)}
		rv = append(rv, sr)
	}
	ctx["results"] = rv

	r.Ctx.RenderTemplate("results.html", ctx, w)
}
