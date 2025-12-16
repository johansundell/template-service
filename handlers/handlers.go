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
	files := make([]string, len(tmplFile))
	for k, t := range tmplFile {
		if h.useFileSystem {
			files[k] = "./tmpl/" + t
		} else {
			files[k] = "tmpl/" + t
		}
	}
	if h.useFileSystem {
		if withBase {
			files = append(files, "./tmpl/base.html")
		}
		//fmt.Println(files)
		return template.ParseFiles(files...)
	}
	if withBase {
		files = append(files, "tmpl/base.html")
	}
	return template.ParseFS(h.tpls, files...)
}
