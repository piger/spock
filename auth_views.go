// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"net/http"
)

// isLoggedIn tests whether an incoming request comes from an authenticated user.
func isLoggedIn(r *vRequest) bool {
	loggedin, ok := r.Session.Values["logged_in"]
	if !ok {
		return false
	}
	if rv, ok := loggedin.(bool); ok {
		return rv
	}
	return false
}

func Login(w http.ResponseWriter, r *vRequest) {
	var error bool

	if isLoggedIn(r) {
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

	http.Redirect(w, r.Request, "/?action=login", http.StatusSeeOther)
}
