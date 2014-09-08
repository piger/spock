// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

// +build bundle

// Embed templates and static resources with go-bindata:
// https://github.com/jteeuwen/go-bindata

package spock

import (
	"github.com/elazarl/go-bindata-assetfs"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

func assembleTemplate(name string) (string, error) {
	basetpl := filepath.Join("templates", "base.html")
	extratpl := filepath.Join("templates", "_extra.html")
	thistpl := filepath.Join("templates", name)

	var tpldata []byte
	for _, tpl := range []string{basetpl, extratpl, thistpl} {
		asset, err := Asset(tpl)
		if err != nil {
			return "", err
		}
		tpldata = append(tpldata, asset...)
	}

	return string(tpldata), nil
}

func LoadTemplate(funcMap *template.FuncMap, name string) *template.Template {
	tpldata, err := assembleTemplate(name)
	if err != nil {
		log.Fatalf("Error assembling template %s: %s\n", name, err)
	}
	t := template.Must(template.New(name).Funcs(*funcMap).Parse(tpldata))
	return t
}

func SetupStaticRoute(prefix, path string) {
	http.Handle(staticPrefix,
		http.StripPrefix(staticPrefix,
			http.FileServer(&assetfs.AssetFS{Asset, AssetDir, "static"})))
}
