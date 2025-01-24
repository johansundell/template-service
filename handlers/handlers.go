package handlers

import (
	"io/fs"
	"text/template"

	"github.com/johansundell/template-service/store"
)

type Handler struct {
	store            *store.Storage
	useFileSystem    bool
	tpls             fs.FS
	nameOfService    string
	versionOfService string
}

func NewHandler(s *store.Storage, ufs bool, f fs.FS, name, version string) *Handler {
	return &Handler{
		store:            s,
		useFileSystem:    ufs,
		tpls:             f,
		nameOfService:    name,
		versionOfService: version,
	}
}

func (h *Handler) getTemplate(withBase bool, tmplFile ...string) (*template.Template, error) {
	for k, t := range tmplFile {
		if h.useFileSystem {
			t = "./tmpl/" + t
		} else {
			t = "tmpl/" + t
		}
		tmplFile[k] = t
	}
	if h.useFileSystem {
		if withBase {
			tmplFile = append(tmplFile, "./tmpl/base.html")
		}
		//fmt.Println(tmplFile)
		return template.ParseFiles(tmplFile...)
	}
	if withBase {
		tmplFile = append(tmplFile, "tmpl/base.html")
	}
	return template.ParseFS(h.tpls, tmplFile...)
}
