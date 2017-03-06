package collection

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	log "github.com/Sirupsen/logrus"
	tid "github.com/Financial-Times/transactionid-utils-go"
)

type NeoHttpHandler interface {
	PutHandler(w http.ResponseWriter, req *http.Request)
	DeleteHandler(w http.ResponseWriter, req *http.Request)
	GetHandler(w http.ResponseWriter, req *http.Request)
	CountHandler(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	collectionService       Service
	collectionType 		string
	relationType 		string
}

func NewNeoHttpHandler(cypherRunner neoutils.NeoConnection, collectionType string, relationType string) NeoHttpHandler {
	newService := NewContentCollectionService(cypherRunner)
	newService.Initialise()

	return &handler{newService, collectionType, relationType}
}

func (hh *handler) PutHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	w.Header().Add("Content-Type", "application/json")

	var body io.Reader = req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		unzipped, err := gzip.NewReader(req.Body)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest, req, uuid)
			return
		}
		defer unzipped.Close()
		body = unzipped
	}

	dec := json.NewDecoder(body)
	inst, docUUID, err := hh.collectionService.DecodeJSON(dec)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest, req, uuid)
		return
	}

	if docUUID != uuid {
		writeJSONError(w, fmt.Sprintf("uuid does not match: '%v' '%v'", docUUID, uuid), http.StatusBadRequest, req, uuid)
		return
	}

	err = hh.collectionService.Write(inst, hh.collectionType, hh.relationType)

	if err != nil {
		switch e := err.(type) {
		case noContentReturnedError:
			writeJSONError(w, e.NoContentReturnedDetails(), http.StatusNoContent, req, uuid)
			return
		case rwapi.ConstraintOrTransactionError:
			writeJSONError(w, e.Error(), http.StatusConflict, req, uuid)
			return
		case invalidRequestError:
			writeJSONError(w, e.InvalidRequestDetails(), http.StatusBadRequest, req, uuid)
			return
		default:
			writeJSONError(w, err.Error(), http.StatusServiceUnavailable, req, uuid)
			return
		}
	}
	//Not necessary for a 200 to be returned, but for PUT requests, if don't specify, don't see 200 status logged in request logs
	w.WriteHeader(http.StatusOK)
}

func (hh *handler) DeleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	deleted, err := hh.collectionService.Delete(uuid, hh.relationType)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable, req, uuid)
		return
	}

	if deleted {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (hh *handler) GetHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := hh.collectionService.Read(uuid, hh.collectionType, hh.relationType)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable, req, uuid)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(obj); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError, req, uuid)
		return
	}
}

func (hh *handler) CountHandler(w http.ResponseWriter, r *http.Request) {

	count, err := hh.collectionService.Count(hh.collectionType)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable, r, "N/A")
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(count); err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable, r, "N/A")
		return
	}
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int, req *http.Request, uuid string) {
	log.WithFields(log.Fields{
		"event":          "error",
		"request_url":    req.URL.String(),
		"transaction_id": req.Header.Get(tid.TransactionIDHeader),
		"status":         statusCode,
		"uuid":           uuid,
	}).Error(errorMsg)

	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
}