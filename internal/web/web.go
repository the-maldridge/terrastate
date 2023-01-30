package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
)

func init() {
	chi.RegisterMethod("LOCK")
	chi.RegisterMethod("UNLOCK")
}

// A Store is an abstraction to a persistent storage system that
// terrastate will use to durably store the state data it is entrusted
// with.  This may very well be a remote KV store, in which case
// terrastate is providing AAA services on top of the existing KV
// store.
type Store interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
	Del([]byte) error

	Close() error
	Sync() error
}

// An Auth component is able to validate a username and password and returns a nil error only if the
type Auth interface {
	AuthUser(context.Context, string, string, string) error
}

// strong type key for context map
type ctxUser struct{}

// Server is an abstraction over all methods needed to operate the
// state server.  It includes the required Store and binds all HTTP
// methods to the appropriate routes.
type Server struct {
	store Store
	l     hclog.Logger

	r chi.Router
	n *http.Server
	a Auth
}

// New returns an initialized server, but not one that is prepared to
// serve.  The embedded echo.Echo instance's Serve method must still
// be called.
func New(opts ...Option) (*Server, error) {
	x := new(Server)
	x.r = chi.NewRouter()
	x.n = &http.Server{}
	x.l = hclog.NewNullLogger()

	for _, o := range opts {
		if err := o(x); err != nil {
			return nil, err
		}
	}

	x.r.Use(middleware.Heartbeat("/healthz"))

	x.r.Route("/state/{project}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proj := chi.URLParam(r, "project")
				u, p, ok := r.BasicAuth()
				if !ok {
					x.l.Debug("Request without basic auth", "project", proj)
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, "HTTP Basic Authentication Required")
					return
				}

				if err := x.a.AuthUser(r.Context(), u, p, proj); err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, "HTTP Basic Authentication Required")
					return
				}

				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser{}, u)))
			})
		})

		r.Get("/{id}", x.getState)
		r.Post("/{id}", x.putState)
		r.Delete("/{id}", x.delState)
		r.Method("LOCK", "/{id}", http.HandlerFunc(x.lockState))
		r.Method("UNLOCK", "/{id}", http.HandlerFunc(x.unlockState))
	})
	return x, nil
}

// Serve binds and serves http on the bound socket.  An error will be
// returned if the server cannot initialize.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	s.n.Addr = bind
	s.n.Handler = s.r
	return s.n.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.n.Shutdown(ctx)
}

func (s *Server) jsonError(w http.ResponseWriter, r *http.Request, rc int, err error) {
	w.WriteHeader(rc)
	enc := json.NewEncoder(w)
	err = enc.Encode(struct {
		Error error
	}{
		Error: err,
	})
	if err != nil {
		fmt.Fprintf(w, "Error writing json error response: %v", err)
	}
}

// getState fetches state for a given id and returns it to the caller.
func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	proj := chi.URLParam(r, "project")
	id := chi.URLParam(r, "id")

	state, err := s.store.Get([]byte(path.Join(proj, id)))
	if err != nil {
		s.l.Error("Error retrieving state", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.l.Info("State Provided", "project", proj, "id", id, "user", r.Context().Value(ctxUser{}))
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(state)
}

func (s *Server) putState(w http.ResponseWriter, r *http.Request) {
	proj := chi.URLParam(r, "project")
	id := chi.URLParam(r, "id")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.l.Error("Error decoding request", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Put([]byte(path.Join(proj, id)), body); err != nil {
		s.l.Error("Error putting state", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Sync(); err != nil {
		s.l.Error("Error flushing state buffers", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.l.Info("State Updated", "project", proj, "id", id, "user", r.Context().Value(ctxUser{}))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) delState(w http.ResponseWriter, r *http.Request) {
	proj := chi.URLParam(r, "project")
	id := chi.URLParam(r, "id")

	if err := s.store.Del([]byte(path.Join(proj, id))); err != nil {
		s.l.Error("Error purging state", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Sync(); err != nil {
		s.l.Error("Error flushing state buffers", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.l.Info("State Purged", "project", proj, "id", id, "user", r.Context().Value(ctxUser{}))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) lockState(w http.ResponseWriter, r *http.Request) {
	proj := chi.URLParam(r, "project")
	id := chi.URLParam(r, "id")

	// In the case of a nil error it must be assumed that a lock
	// is being held.
	if l, err := s.store.Get([]byte(path.Join(proj, id, "lock"))); err == nil && l != nil {
		s.l.Warn("Could not aquire lock, already held", "project", proj, "id", id)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked)
		w.Write(l)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.l.Error("Error decoding request", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Put([]byte(path.Join(proj, id, "lock")), body); err != nil {
		s.l.Error("Error putting state", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Sync(); err != nil {
		s.l.Error("Error flushing state buffers", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.l.Info("State Locked", "project", proj, "id", id, "user", r.Context().Value(ctxUser{}))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) unlockState(w http.ResponseWriter, r *http.Request) {
	proj := chi.URLParam(r, "project")
	id := chi.URLParam(r, "id")

	if err := s.store.Del([]byte(path.Join(proj, id, "lock"))); err != nil {
		s.l.Error("Error releasing lock", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	if err := s.store.Sync(); err != nil {
		s.l.Error("Error flushing state buffers", "project", proj, "id", id, "error", err)
		s.jsonError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.l.Info("State Unlocked", "project", proj, "id", id, "user", r.Context().Value(ctxUser{}))
	w.WriteHeader(http.StatusOK)
}
