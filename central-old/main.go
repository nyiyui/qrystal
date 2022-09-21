package central

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nyiyui/qanms/node"
)

type Server struct {
	r  *mux.Router
	c  Config
	cc node.CentralConfig
}
type Config struct {
}

func (s *Server) combined(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.UserAgent(), "QANMSClient") {
			http.Error(w, "only usable by QANMSClient", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func New(c Config) (*Server, error) {
	s := &Server{
		r: mux.NewRouter(),
		c: c,
	}
	s.r.Methods("GET").Path("/v1/get").Handler(s.combined(http.HandlerFunc(s.handleGet)))
	return s, nil
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(s.cc)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(w, buf)
	if err != nil {
		return
	}
	return
}
