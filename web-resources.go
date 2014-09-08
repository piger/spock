// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

// +build !bundle

// Load templates and static resources from the filesystem.

package spock

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func LoadTemplate(funcMap *template.FuncMap, name string) *template.Template {
	baseTemplate := filepath.Join(DataDir, "templates", "base.html")
	extraTemplate := filepath.Join(DataDir, "templates", "_extra.html")
	t := template.Must(template.New(name).Funcs(*funcMap).ParseFiles(filepath.Join(DataDir, "templates", name), baseTemplate, extraTemplate))
	return t
}

func SetupStaticRoute(prefix, path string) {
	http.Handle(prefix,
		http.StripPrefix(prefix,
			http.FileServer(http.Dir(path))))
}
