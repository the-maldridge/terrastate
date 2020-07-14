package web

import (
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/labstack/echo"
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
func New(kv Store) (*Server, error) {
	e := echo.New()
	x := new(Server)
	x.Echo = e
	x.store = kv

	x.GET("/state/:id", x.getState)
	x.POST("/state/:id", x.putState)
	x.DELETE("/state/:id", x.delState)

	return x, nil
}

// getState fetches state for a given id and returns it to the caller.
func (s *Server) getState(c echo.Context) error {
	id := c.Param("id")

	state, err := s.store.Get([]byte(id))
	if err != nil {
		s.l.Error("Error retrieving state", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.Blob(http.StatusOK, "text", state)
}

func (s *Server) putState(c echo.Context) error {
	id := c.Param("id")

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		s.l.Error("Error decoding request", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if err := s.store.Put([]byte(id), body); err != nil {
		s.l.Error("Error putting state", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) delState(c echo.Context) error {
	id := c.Param("id")

	if err := s.store.Del([]byte(id)); err != nil {
		s.l.Error("Error purging state", "id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusOK)
}
