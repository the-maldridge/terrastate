package web

import (
	"context"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/hashicorp/go-hclog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

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

// Server is an abstraction over all methods needed to operate the
// state server.  It includes the required Store and binds all HTTP
// methods to the appropriate routes.
type Server struct {
	*echo.Echo

	store Store
	l     hclog.Logger
}

// New returns an initialized server, but not one that is prepared to
// serve.  The embedded echo.Echo instance's Serve method must still
// be called.
func New(kv Store, auth Auth) *Server {
	e := echo.New()
	e.Logger.SetLevel(99)
	x := new(Server)
	x.Echo = e
	x.store = kv

	x.GET("/alive", func(c echo.Context) error { return c.String(http.StatusOK, "OK") })

	sg := x.Group("/state")

	sg.Use(middleware.BasicAuth(func(u, p string, c echo.Context) (bool, error) {
		proj := c.Param("project")
		if err := auth.AuthUser(c.Request().Context(), u, p, proj); err != nil {
			return false, err
		}
		c.Set("user", u)
		return true, nil
	}))

	sg.GET("/:project/:id", x.getState)
	sg.POST("/:project/:id", x.putState)
	sg.DELETE("/:project/:id", x.delState)
	sg.PUT("/:project/:id/lock", x.lockState)
	sg.DELETE("/:project/:id/lock", x.unlockState)

	return x
}

// SetLogger sets the logger for the top level of the web system.
func (s *Server) SetLogger(l hclog.Logger) {
	s.l = l
}

// getState fetches state for a given id and returns it to the caller.
func (s *Server) getState(c echo.Context) error {
	proj := c.Param("project")
	id := c.Param("id")

	state, err := s.store.Get([]byte(path.Join(proj, id)))
	if err != nil {
		s.l.Error("Error retrieving state", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	s.l.Info("State Provided", "project", proj, "id", id, "user", c.Get("user"))
	return c.Blob(http.StatusOK, "application/json", state)
}

func (s *Server) putState(c echo.Context) error {
	proj := c.Param("project")
	id := c.Param("id")

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		s.l.Error("Error decoding request", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if err := s.store.Put([]byte(path.Join(proj, id)), body); err != nil {
		s.l.Error("Error putting state", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if err := s.store.Sync(); err != nil {
		s.l.Error("Error flushing state buffers", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	s.l.Info("State Updated", "project", proj, "id", id, "user", c.Get("user"))
	return c.NoContent(http.StatusOK)
}

func (s *Server) delState(c echo.Context) error {
	proj := c.Param("project")
	id := c.Param("id")

	if err := s.store.Del([]byte(path.Join(proj, id))); err != nil {
		s.l.Error("Error purging state", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	s.l.Info("State Purged", "project", proj, "id", id, "user", c.Get("user"))
	return c.NoContent(http.StatusOK)
}

func (s *Server) lockState(c echo.Context) error {
	proj := c.Param("project")
	id := c.Param("id")

	// In the case of a nil error it must be assumed that a lock
	// is being held.
	if l, err := s.store.Get([]byte(path.Join(proj, id, "lock"))); err == nil && l != nil {
		s.l.Warn("Could not aquire lock, already held", "project", proj, "id", id)
		return c.Blob(http.StatusConflict, "application/json", l)
	}

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		s.l.Error("Error decoding request", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if err := s.store.Put([]byte(path.Join(proj, id, "lock")), body); err != nil {
		s.l.Error("Error putting state", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	s.l.Info("State Locked", "project", proj, "id", id, "user", c.Get("user"))
	return c.NoContent(http.StatusOK)
}

func (s *Server) unlockState(c echo.Context) error {
	proj := c.Param("project")
	id := c.Param("id")

	if err := s.store.Del([]byte(path.Join(proj, id, "lock"))); err != nil {
		s.l.Error("Error releasing lock", "project", proj, "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	s.l.Info("State Unlocked", "project", proj, "id", id, "user", c.Get("user"))
	return c.NoContent(http.StatusOK)
}
