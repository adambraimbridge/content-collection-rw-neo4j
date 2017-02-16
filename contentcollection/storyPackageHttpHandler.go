package contentcollection

import (
	"net/http"
)

type storyPackageHttpHandler struct {
	handler NeoHttpHandler
}

func NewStoryPackageHttpHandler(handler NeoHttpHandler) storyPackageHttpHandler {
	return storyPackageHttpHandler{handler}
}

func (hh *storyPackageHttpHandler) GetHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.GetHandler(w, req, "StoryPackage")
}

func (hh *storyPackageHttpHandler) PutHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.PutHandler(w, req, "StoryPackage")
}

func (hh *storyPackageHttpHandler) DeleteHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.DeleteHandler(w, req)
}

func (hh *storyPackageHttpHandler) CountHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.CountHandler(w, req, "StoryPackage")
}


