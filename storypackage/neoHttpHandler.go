package storypackage

import (
	"encoding/json"
	"fmt"
	"net/http"

	"compress/gzip"

	"io"

	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	"github.com/gorilla/mux"
)


type NeoHttpHandler interface {
	PutHandler(w http.ResponseWriter, req *http.Request, collectionType string)
	DeleteHandler(w http.ResponseWriter, req *http.Request)
	GetHandler(w http.ResponseWriter, req *http.Request, collectionType string)
	CountHandler(w http.ResponseWriter, r *http.Request, collectionType string)
}


type handler struct {
	s Service
}

func NewNeoHttpHandler(cypherRunner neoutils.NeoConnection) NeoHttpHandler {
	service := NewCypherStoryPackageService(cypherRunner)
	service.Initialise()
	return &handler{service}
}

func (hh *handler) PutHandler(w http.ResponseWriter, req *http.Request, collectionType string) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	w.Header().Add("Content-Type", "application/json")

	var body io.Reader = req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		unzipped, err := gzip.NewReader(req.Body)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer unzipped.Close()
		body = unzipped
	}

	dec := json.NewDecoder(body)
	inst, docUUID, err := hh.s.DecodeJSON(dec)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if docUUID != uuid {
		writeJSONError(w, fmt.Sprintf("uuid does not match: '%v' '%v'", docUUID, uuid), http.StatusBadRequest)
		return
	}

	err = hh.s.Write(inst, collectionType)

	if err != nil {
		switch e := err.(type) {
		case noContentReturnedError:
			writeJSONError(w, e.NoContentReturnedDetails(), http.StatusNoContent)
			return
		case *neoutils.ConstraintViolationError:
			// TODO: remove neo specific error check once all apps are
			// updated to use neoutils.Connect() because that maps errors
			// to rwapi.ConstraintOrTransactionError
			writeJSONError(w, e.Error(), http.StatusConflict)
			return
		case rwapi.ConstraintOrTransactionError:
			writeJSONError(w, e.Error(), http.StatusConflict)
			return
		case invalidRequestError:
			writeJSONError(w, e.InvalidRequestDetails(), http.StatusBadRequest)
			return
		default:
			writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	//Not necessary for a 200 to be returned, but for PUT requests, if don't specify, don't see 200 status logged in request logs
	w.WriteHeader(http.StatusOK)
}

func (hh *handler) DeleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	deleted, err := hh.s.Delete(uuid)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if deleted {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (hh *handler) GetHandler(w http.ResponseWriter, req *http.Request, collectionType string) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := hh.s.Read(uuid, collectionType)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(obj); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (hh *handler) CountHandler(w http.ResponseWriter, r *http.Request, collectionType string) {

	count, err := hh.s.Count(collectionType)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(count); err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
}

