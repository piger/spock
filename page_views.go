package spock

import (
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"time"
)

func getPagePath(r *vRequest) string {
	vars := mux.Vars(r.Request)
	return vars["pagepath"]
}

func ShowPage(w http.ResponseWriter, r *vRequest) {
	ctx := newTemplateContext(r)

	renderStart := time.Now()

	pagepath := getPagePath(r)
	page, err := (*r.Ctx.Storage).LookupPage(pagepath)
	if page == nil && err == nil {
		http.NotFound(w, r.Request)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	page, err := (*r.Ctx.Storage).LookupPage(pagepath)
	if page == nil && err == nil {
		http.NotFound(w, r.Request)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

		http.Redirect(w, r.Request, page.ShortName(), http.StatusSeeOther)
		return
	}

	ctx := newTemplateContext(r)
	ctx["content"] = template.HTML(page.RawBytes)
	ctx["pageName"] = page.Path
	ctx["isNew"] = false
	ctx["comment"] = ""

	r.Ctx.RenderTemplate("edit_page.html", ctx, w)
}

func LookupAuthor(r *vRequest) (fullname, email string) {
	if ifullname, ok := r.Session.Values["name"]; !ok {
		fullname = "Anonymous"
	} else {
		fullname = ifullname.(string)
	}

	if iemail, ok := r.Session.Values["email"]; !ok {
		email = "anonymous@wiki.int"
	} else {
		email = iemail.(string)
	}

	return
}
