package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/spikeekips/sebak/lib/network"
)

func TestFindme(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, strings.Repeat("findme", 50))
	}
}

func TestEvent(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//if r.Method != "GET" {
		//	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		//	return
		//}

		flusher, _ := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		//w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher.Flush()

		ticker := time.NewTicker(500 * time.Millisecond)

		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			ticker.Stop()
		}()

		for _ = range ticker.C {
			w.Write([]byte(fmt.Sprintf("%s: %s", time.Now().String(), strings.Repeat("findme", 500))))
			flusher.Flush()
		}
	}
}

func TestMessage(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); strings.ToLower(ct) != "application/json" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}

		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		w.Write(body)
		return
	}
}
