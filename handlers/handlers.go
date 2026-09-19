package handlers

import (
	"io/fs"
	"path/filepath"
	"text/template"

	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/utils"
)

type Handler struct {
	store            store.Store
	useFileSystem    bool
	tpls             fs.FS
	nameOfService    string
	versionOfService string
}

func NewHandler(s store.Store, ufs bool, f fs.FS, name, version string) *Handler {
	return &Handler{
		store:            s,
		useFileSystem:    ufs,
		tpls:             f,
		nameOfService:    name,
		versionOfService: version,
	}
}

func (h *Handler) getTemplate(withBase bool, tmplFile ...string) (*template.Template, error) {
	basePath := utils.GetBinaryBasePath()
	files := make([]string, len(tmplFile))
	for k, t := range tmplFile {
		if h.useFileSystem {
			files[k] = filepath.Join(basePath, "tmpl", t)
		} else {
			files[k] = "tmpl/" + t
		}
	}
	if h.useFileSystem {
		if withBase {
			files = append(files, filepath.Join(basePath, "tmpl", "base.html"))
		}
		return template.ParseFiles(files...)
	}
	if withBase {
		files = append(files, "tmpl/base.html")
	}
	return template.ParseFS(h.tpls, files...)
}
