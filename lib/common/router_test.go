package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRouterHeaderMatcher(t *testing.T) {
	router := mux.NewRouter()

	dummyHandler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		return
	}

	router.HandleFunc("/showme", dummyHandler).MatcherFunc(PostAndJSONMatcher)
	server := httptest.NewServer(router)
	u, _ := url.Parse(server.URL)
	u.Path = "/showme"

	{ // GET msut be passed
		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{ // POST && empty 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	{ // POST && invalid 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		req.Header.Set("Content-Type", "text/plain")
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	{ // POST && valid 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestRouterHeaderMatcherWithMethodMatcher(t *testing.T) {
	router := mux.NewRouter()

	dummyHandler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		return
	}

	router.HandleFunc("/showme", dummyHandler).Methods("GET", "POST").MatcherFunc(PostAndJSONMatcher)
	server := httptest.NewServer(router)
	u, _ := url.Parse(server.URL)
	u.Path = "/showme"

	{ // GET msut be passed
		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{ // POST && empty 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	{ // POST && invalid 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		req.Header.Set("Content-Type", "text/plain")
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	{ // POST && valid 'Content-Type'
		req, _ := http.NewRequest("POST", u.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
