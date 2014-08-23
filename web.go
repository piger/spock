package spock

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

var (
	sessionName  = "vandine-session"
	templatesDir = "./data/templates"
	staticDir    = "./data/static"
	staticPrefix = "/static/"
)

// web
type AppContext struct {
	SessionStore sessions.Store
	Storage      *Storage
	Templates    map[string]*template.Template
	XsrfSecret   string
}

type vRequest struct {
	Ctx     *AppContext
	Session *sessions.Session
	Request *http.Request
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
			Ctx:     ac,
			Session: session,
			Request: r,
		}
		h.ServeHTTP(w, &vreq)
	})
}

func (ac *AppContext) RenderTemplate(name string, context map[string]interface{}, w http.ResponseWriter) {
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

func Index(w http.ResponseWriter, r *vRequest) {
	context := make(map[string]interface{})
	context["foo"] = "bar"

	r.Ctx.RenderTemplate("index.html", context, w)
}

func Login(w http.ResponseWriter, r *vRequest) {
	context := make(map[string]interface{})
	r.Ctx.RenderTemplate("login.html", context, w)
}

func LoginPOST(w http.ResponseWriter, r *vRequest) {
	http.Redirect(w, r.Request, "/login", 302)
}

func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"formatDatetime": formatDatetime,
		"gravatarHash":   gravatarHash,
	}

	templateNames := []string{
		"edit_page.html",
		"log.html",
		"login.html",
		"ls.html",
		"page.html",
		"rename.html",
		"results.html",
	}
	baseTemplate := filepath.Join(templatesDir, "base.html")
	for _, tplName := range templateNames {
		t := template.Must(template.New(tplName).Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, tplName), baseTemplate))
		templates[tplName] = t
	}
	return templates
}

func RunServer(address string, storage Storage) error {
	ac := &AppContext{
		SessionStore: sessions.NewCookieStore([]byte("lalala")),
		Storage:      &storage,
		Templates:    loadTemplates(),
	}

	http.Handle(staticPrefix,
		http.StripPrefix(staticPrefix,
			http.FileServer(http.Dir(staticDir))))

	r := mux.NewRouter()
	r.Handle("/", WithRequest(ac, vHandlerFunc(Index))).Name("index")
	r.Handle("/login", WithRequest(ac, vHandlerFunc(Login))).Name("login").Methods("GET")
	r.Handle("/login", WithRequest(ac, vHandlerFunc(LoginPOST))).Name("login").Methods("POST")
	http.Handle("/", r)

	log.Printf("Listening on %s\n", address)
	err := http.ListenAndServe(address, nil)
	return err
}
