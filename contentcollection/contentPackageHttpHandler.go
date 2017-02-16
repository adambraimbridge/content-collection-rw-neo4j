package contentcollection

import (
	"net/http"
)

type contentPackagePackageHttpHandler struct {
	handler NeoHttpHandler
}

func NewContentPackageHttpHandler(handler NeoHttpHandler) storyPackageHttpHandler {
	return storyPackageHttpHandler{handler}
}

func (hh *contentPackagePackageHttpHandler) GetHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.GetHandler(w, req, "ContentPackage")
}

func (hh *contentPackagePackageHttpHandler) PutHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.PutHandler(w, req, "ContentPackage")
}

func (hh *contentPackagePackageHttpHandler) DeleteHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.DeleteHandler(w, req)
}

func (hh *contentPackagePackageHttpHandler) CountHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.CountHandler(w, req, "ContentPackage")
}



