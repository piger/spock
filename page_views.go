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
