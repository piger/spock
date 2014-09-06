// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"
)

var (
	sessionName  = "vandine-session"
	staticPrefix = "/static/"

	// DataDir is the directory containing HTML templates, static files and other
	// runtime resources; this is exported so you can configure it from the cmd launcher.
	// Unfortunately this is used by the RenderRst method in page.go; we should
	// find a better way to handle this...
	DataDir = "./data"
)

// User is a representation of a wiki user.
type User struct {
	Authenticated bool
	Name          string
	Email         string
}

// Alert is used to show informational messages in the web GUI.
type Alert struct {
	Level   string
	Message string
}

// pageCache is a silly cache that will hold the rendered version of a wiki page.
type pageCache struct {
	lock        sync.RWMutex
	PageRenders map[string][]byte
}

func (pc *pageCache) Get(path string) ([]byte, bool) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	page, ok := pc.PageRenders[path]
	return page, ok
}

func (pc *pageCache) Set(path string, html []byte) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	pc.PageRenders[path] = html
}

func (pc *pageCache) Flush(path string) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	delete(pc.PageRenders, path)
}

// PageCache contains the rendering cache.
var PageCache *pageCache

func init() {
	gob.Register(&Alert{})
	PageCache = &pageCache{PageRenders: make(map[string][]byte)}
}

func UserFromSession(session *sessions.Session) *User {
	user := &User{}
	if loggedIn, ok := session.Values["logged_in"]; ok {
		user.Authenticated = loggedIn.(bool)
	} else {
		user.Authenticated = false
	}

	if username, ok := session.Values["name"]; ok {
		user.Name = username.(string)
	} else {
		user.Name = "Anonymous"
	}

	if email, ok := session.Values["email"]; ok {
		user.Email = email.(string)
	} else {
		user.Email = "anon@wiki.int"
	}

	return user
}

type AppContext struct {
	SessionStore sessions.Store
	Storage      Storage
	Templates    map[string]*template.Template
	XsrfSecret   string
	IndexSrv     string
	Index        Index
}

type vRequest struct {
	Ctx      *AppContext
	Session  *sessions.Session
	Request  *http.Request
	AuthUser *User
}

type vHandler interface {
	ServeHTTP(w http.ResponseWriter, r *vRequest)
}

type vHandlerFunc func(w http.ResponseWriter, r *vRequest)

func (f vHandlerFunc) ServeHTTP(w http.ResponseWriter, r *vRequest) {
	f(w, r)
}

func WithRequest(ac *AppContext, h vHandler) http.Handler {
	sessionOpts := sessions.Options{
		HttpOnly: true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := ac.SessionStore.Get(r, sessionName)
		session.Options = &sessionOpts

		vreq := vRequest{
			Ctx:      ac,
			Session:  session,
			Request:  r,
			AuthUser: UserFromSession(session),
		}
		h.ServeHTTP(w, &vreq)
	})
}

type TemplateContext map[string]interface{}

func newTemplateContext(r *vRequest) TemplateContext {
	tc := make(map[string]interface{})
	tc["user"] = r.AuthUser
	return tc
}

func (ac *AppContext) RenderTemplate(name string, context TemplateContext, w http.ResponseWriter) {
	var buf bytes.Buffer
	t, ok := ac.Templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("Cannot find template \"%s\"", name), http.StatusInternalServerError)
		return
	}
	err := t.ExecuteTemplate(&buf, "base", context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// Return a slice of session Alerts; note: will save the current session.
func GetAlerts(r *vRequest, w http.ResponseWriter) []*Alert {
	var alerts []*Alert

	if flashes := r.Session.Flashes(); len(flashes) > 0 {
		for _, flash := range flashes {
			alerts = append(alerts, flash.(*Alert))
		}
	}
	r.Session.Save(r.Request, w)

	return alerts
}

func AddAlert(message, level string, r *vRequest) {
	r.Session.AddFlash(&Alert{Level: level, Message: message})
}

// views

func IndexRedirect(w http.ResponseWriter, r *vRequest) {
	_, exists, err := r.Ctx.Storage.LookupPage("index")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		http.Redirect(w, r.Request, "/index", http.StatusFound)
		return
	}

	ctx := newTemplateContext(r)
	r.Ctx.RenderTemplate("welcome.html", ctx, w)
}

func Login(w http.ResponseWriter, r *vRequest) {
	var error bool

	if loggedIn, ok := r.Session.Values["logged_in"]; ok && loggedIn.(bool) {
		http.Redirect(w, r.Request, "/index", http.StatusSeeOther)
		return
	}

	if r.Request.Method == "POST" {
		if err := r.Request.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		name := r.Request.PostFormValue("name")
		email := r.Request.PostFormValue("email")

		if name != "" && email != "" {
			r.Session.Values["logged_in"] = true
			r.Session.Values["name"] = name
			r.Session.Values["email"] = email
			r.Session.Save(r.Request, w)
			http.Redirect(w, r.Request, "/index", http.StatusSeeOther)
			return
		}

		// else...
		error = true
	}

	ctx := newTemplateContext(r)
	ctx["error"] = error

	r.Ctx.RenderTemplate("login.html", ctx, w)
}

func Logout(w http.ResponseWriter, r *vRequest) {
	delete(r.Session.Values, "logged_in")
	delete(r.Session.Values, "name")
	delete(r.Session.Values, "email")
	r.Session.Save(r.Request, w)

	http.Redirect(w, r.Request, "/login", http.StatusSeeOther)
}

func loadTemplates(router *mux.Router) map[string]*template.Template {
	templates := make(map[string]*template.Template)

	// reverse an URL from inside a template context; needs to access the mux router
	reverse := func(name string, params ...interface{}) string {
		strparams := make([]string, len(params))
		for i, param := range params {
			strparams[i] = fmt.Sprint(param)
		}

		route := router.GetRoute(name)
		if route == nil {
			log.Fatalf("Route %s does not exists", name)
		}
		url, err := route.URL(strparams...)
		if err != nil {
			log.Fatal(err)
		}

		return url.Path
	}

	funcMap := template.FuncMap{
		"formatDatetime": formatDatetime,
		"gravatarHash":   gravatarHash,
		"reverse":        reverse,
	}

	templateNames := []string{
		"edit_page.html",
		"log.html",
		"login.html",
		"ls.html",
		"page.html",
		"rename.html",
		"results.html",
		"diff.html",
		"preview.html",
		"delete.html",
		"welcome.html",
	}
	baseTemplate := filepath.Join(DataDir, "templates", "base.html")
	extraTemplate := filepath.Join(DataDir, "templates", "_extra.html")
	for _, tplName := range templateNames {
		t := template.Must(template.New(tplName).Funcs(funcMap).ParseFiles(filepath.Join(DataDir, "templates", tplName), baseTemplate, extraTemplate))
		templates[tplName] = t
	}
	return templates
}

func RunServer(address string, ac *AppContext) error {
	r := mux.NewRouter()
	ac.Templates = loadTemplates(r)

	http.Handle(staticPrefix,
		http.StripPrefix(staticPrefix,
			http.FileServer(http.Dir(filepath.Join(DataDir, "static")))))

	r.Handle("/", WithRequest(ac, vHandlerFunc(IndexRedirect))).Name("index")
	r.Handle("/login", WithRequest(ac, vHandlerFunc(Login))).Name("login")
	r.Handle("/logout", WithRequest(ac, vHandlerFunc(Logout))).Name("logout")
	r.Handle("/ls", WithRequest(ac, vHandlerFunc(ListPages))).Name("list_pages")
	r.Handle("/search", WithRequest(ac, vHandlerFunc(SearchPages))).Name("search_pages")

	pagePattern := "/{pagepath:[a-zA-Z0-9_/.]+}"
	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(EditPage))).Queries("edit", "1").Name("edit_page")
	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(ShowPageLog))).Queries("log", "1").Name("show_log")
	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(RenamePage))).Queries("rename", "1").Name("rename_page")
	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(DeletePage))).Queries("delete", "1").Name("delete_page")
	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(DiffPage))).Queries("diff", "{sha:[a-zA-Z0-9]{40}}").Name("diff_page")

	r.Handle(pagePattern, WithRequest(ac, vHandlerFunc(ShowPage))).Name("show_page")
	http.Handle("/", r)

	fmt.Printf("Wiki running on http://%s\n", address)
	err := http.ListenAndServe(address, nil)
	return err
}
