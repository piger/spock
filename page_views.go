package spock

import (
	"github.com/gorilla/mux"
	"html/template"
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
		http.NotFound(w, r.Request)
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

	ctx["content"] = template.HTML(html)
	ctx["author_email"] = lastlog.Email
	ctx["author_name"] = lastlog.Name
	ctx["modification_time"] = lastlog.When
	ctx["last_commit"] = lastlog.Message
	ctx["render_time"] = time.Since(renderStart)

	r.Ctx.RenderTemplate("page.html", ctx, w)
}

func EditPage(w http.ResponseWriter, r *vRequest) {
	pagepath := getPagePath(r)
	page, _, err := (*r.Ctx.Storage).LookupPage(pagepath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// else if !exists {
	//		http.NotFound(w, r.Request)
	//		return
	//	}

	if r.Request.Method == "POST" {
		if err := r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		content := r.Request.PostFormValue("content")
		comment := r.Request.PostFormValue("comment")
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
	ctx["content"] = template.HTML(page.RawBytes)
	ctx["pageName"] = page.ShortName()
	ctx["isNew"] = false
	ctx["comment"] = ""

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
			comment = "(no comment)"
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
	_, err = (*r.Ctx.Storage).DiffPage(page, shaParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// INSERT CODE HERE
}
