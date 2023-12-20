package server

import (
	"net/http"

	"github.com/gomatbase/go-we"
)

func dashboard(writer we.ResponseWriter, scope we.RequestScope) error {
	writer.WriteHeader(http.StatusFound)
	_, e := writer.Write([]byte("Soon to be dashboard!"))
	return e
}
