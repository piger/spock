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
type User struct {
	Authenticated bool
	Name          string
	Email         string
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
	Storage      *Storage
	Templates    map[string]*template.Template
	XsrfSecret   string
	IndexSrv     string
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

func newTemplateContext(r *vRequest) (tc map[string]interface{}) {
	tc = make(map[string]interface{})
	tc["user"] = r.AuthUser
	return
}

// views

func Index(w http.ResponseWriter, r *vRequest) {
	http.Redirect(w, r.Request, "/index", http.StatusFound)
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
		} else {
			error = true
		}
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
	}
	baseTemplate := filepath.Join(templatesDir, "base.html")
	for _, tplName := range templateNames {
		t := template.Must(template.New(tplName).Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, tplName), baseTemplate))
		templates[tplName] = t
	}
	return templates
}

func RunServer(address string, storage Storage, indexSrv string) error {
	r := mux.NewRouter()

	ac := &AppContext{
		SessionStore: sessions.NewCookieStore([]byte("lalala")),
		XsrfSecret:   "lalala",
		Storage:      &storage,
		Templates:    loadTemplates(r),
		IndexSrv:     indexSrv,
	}

	http.Handle(staticPrefix,
		http.StripPrefix(staticPrefix,
			http.FileServer(http.Dir(staticDir))))

	r.Handle("/", WithRequest(ac, vHandlerFunc(Index))).Name("index")
	r.Handle("/login", WithRequest(ac, vHandlerFunc(Login))).Name("login")
	r.Handle("/logout", WithRequest(ac, vHandlerFunc(Logout))).Name("logout")
	r.Handle("/ls", WithRequest(ac, vHandlerFunc(ListPages))).Name("list_pages")
	r.Handle("/search", WithRequest(ac, vHandlerFunc(SearchPages))).Name("search_pages")

	r.Handle("/{pagepath:[a-zA-Z0-9_/.]+}", WithRequest(ac, vHandlerFunc(EditPage))).Queries("edit", "1").Name("edit_page")
	r.Handle("/{pagepath:[a-zA-Z0-9_/.]+}", WithRequest(ac, vHandlerFunc(ShowPageLog))).Queries("log", "1").Name("show_log")
	r.Handle("/{pagepath:[a-zA-Z0-9_/.]+}", WithRequest(ac, vHandlerFunc(RenamePage))).Queries("rename", "1").Name("rename_page")
	r.Handle("/{pagepath:[a-zA-Z0-9_/.]+}", WithRequest(ac, vHandlerFunc(DiffPage))).Queries("diff", "{sha:[a-zA-Z0-9]{40}}").Name("diff_page")

	r.Handle("/{pagepath:[a-zA-Z0-9_/.]+}", WithRequest(ac, vHandlerFunc(ShowPage))).Name("show_page")
	http.Handle("/", r)

	log.Printf("Listening on: %s\n", address)
	err := http.ListenAndServe(address, nil)
	return err
}
