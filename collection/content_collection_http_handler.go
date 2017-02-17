package collection

import (
	"net/http"
)

type contentCollectionHttpHandler struct {
	handler        NeoHttpHandler
	collectionType string
}

func NewContentCollectionHttpHandler(handler NeoHttpHandler, collectionType string) contentCollectionHttpHandler {
	return contentCollectionHttpHandler{handler, collectionType}
}

func (hh *contentCollectionHttpHandler) GetHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.GetHandler(w, req, hh.collectionType)
}

func (hh *contentCollectionHttpHandler) PutHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.PutHandler(w, req, hh.collectionType)
}

func (hh *contentCollectionHttpHandler) DeleteHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.DeleteHandler(w, req)
}

func (hh *contentCollectionHttpHandler) CountHandler(w http.ResponseWriter, req *http.Request) {
	hh.handler.CountHandler(w, req, hh.collectionType)
}


