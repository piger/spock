// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
)

var (
	sessionName     = "spock-session"
	staticPrefix    = "/static/"
	sessionDuration = 86400 * 7 // 7 days
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

func init() {
	gob.Register(&Alert{})
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
	Router       *mux.Router
	Templates    map[string]*template.Template
	XsrfSecret   string
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
		Path:     "/",
		MaxAge:   sessionDuration,
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

	templateBox, err := rice.FindBox("data/templates")
	if err != nil {
		log.Fatal(err)
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
		"delete.html",
		"welcome.html",
	}
	for _, tplName := range templateNames {
		templates[tplName] = LoadRiceTemplate(tplName, &funcMap, templateBox)
	}
	return templates
}

func LoadRiceTemplate(name string, funcMap *template.FuncMap, box *rice.Box) *template.Template {
	tpl := box.MustBytes(name)
	tpl = append(tpl, box.MustBytes("base.html")...)
	tpl = append(tpl, box.MustBytes("_extra.html")...)
	t := template.Must(template.New(name).Funcs(*funcMap).Parse(string(tpl)))
	return t
}

func RunServer(address string, ac *AppContext) error {
	r := mux.NewRouter()
	ac.Templates = loadTemplates(r)
	ac.Router = r

	http.Handle(staticPrefix, http.StripPrefix(staticPrefix, http.FileServer(rice.MustFindBox("data/static").HTTPBox())))

	r.Handle("/", WithRequest(ac, vHandlerFunc(IndexRedirect))).Name("index")
	r.Handle("/login", WithRequest(ac, vHandlerFunc(Login))).Name("login")
	r.Handle("/logout", WithRequest(ac, vHandlerFunc(Logout))).Name("logout")
	r.Handle("/ls", WithRequest(ac, vHandlerFunc(IndexAllPages))).Queries("action", "index")
	r.Handle("/ls", WithRequest(ac, vHandlerFunc(ListPages))).Name("list_pages")
	r.Handle("/search", WithRequest(ac, vHandlerFunc(SearchPages))).Name("search_pages")
	r.Handle(`/{filename:.*?\.(png|jpe?g|bmp|gif|pdf)$}`, WithRequest(ac, vHandlerFunc(ServeFile))).Name("serve_file")

	pp := `/{pagepath:[a-zA-Z0-9_/.-]+}`
	r.Handle(pp, WithRequest(ac, vHandlerFunc(EditPage))).Queries("action", "edit").Name("edit_page")
	r.Handle(pp, WithRequest(ac, vHandlerFunc(ShowPageLog))).Queries("action", "log").Name("show_log")
	r.Handle(pp, WithRequest(ac, vHandlerFunc(RenamePage))).Queries("action", "rename").Name("rename_page")
	r.Handle(pp, WithRequest(ac, vHandlerFunc(DeletePage))).Queries("action", "delete").Name("delete_page")
	r.Handle(pp, WithRequest(ac, vHandlerFunc(DiffPage))).Queries("action", "diff", "startrev", `{startrev:[a-zA-Z0-9]{40}}`, "endrev", `{endrev:[a-zA-Z0-9]{40}}`).Name("diff_page")

	r.Handle(pp, WithRequest(ac, vHandlerFunc(ShowPage))).Name("show_page")
	http.Handle("/", r)

	fmt.Printf("Wiki running on http://%s\n", address)
	err := http.ListenAndServe(address, nil)
	return err
}
