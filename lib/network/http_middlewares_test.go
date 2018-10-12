package network

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRecoverMiddleware(t *testing.T) {
	handlerURL := UrlPathPrefixAPI + "/test"
	panicMsg := "Don't panic,just use go"
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(panicMsg)
	}

	router := mux.NewRouter()
	router.Use(RecoverMiddleware(nil))
	router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + handlerURL)
	require.Nil(t, err)

	require.Equal(t, 500, resp.StatusCode)
	require.Equal(t, "application/problem+json", resp.Header["Content-Type"][0])

	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.Nil(t, err)

	var msg map[string]interface{}
	err = json.Unmarshal(bs, &msg)
	require.Nil(t, err)
	require.Equal(t, "panic: "+panicMsg, msg["title"])
}
