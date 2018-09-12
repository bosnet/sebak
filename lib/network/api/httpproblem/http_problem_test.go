package httpproblem

import (
	"boscoin.io/sebak/lib/error"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProblem(t *testing.T) {

	router := mux.NewRouter()

	statusProblem := NewStatusProblem(http.StatusBadRequest)
	detailedStatusProblem := NewDetailedStatusProblem(http.StatusBadRequest, "paramaters are not enough")
	errorProblem := NewErrorProblem(errors.ErrorInvalidOperation)

	router.HandleFunc("/problem_status_default", func(w http.ResponseWriter, r *http.Request) {
		statusProblem.Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_with_detail", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_with_detail_instance", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.SetInstance("http://boscoin.io/httperror/details/1").Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_default_with_detail", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.Problem(w, "bad request yo!", -1)
	})

	router.HandleFunc("/problem_with_error", func(w http.ResponseWriter, r *http.Request) {
		errorProblem.Problem(w, "", -1)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	// problem_status_default
	{
		url := ts.URL + fmt.Sprintf("/problem_status_default")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := statusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Empty(t, m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_status_with_detail
	{
		url := ts.URL + fmt.Sprintf("/problem_status_with_detail")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, p.Detail, m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_status_with_detail_instance
	{
		url := ts.URL + fmt.Sprintf("/problem_status_with_detail_instance")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem.SetInstance("http://boscoin.io/httperror/details/1")
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, p.Detail, m["detail"])
			require.Equal(t, p.Instance, m["instance"])
		}
	}

	// problem_status_default_with_detail
	{
		url := ts.URL + fmt.Sprintf("/problem_status_default_with_detail")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, "bad request yo!", m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_with_error
	{
		url := ts.URL + fmt.Sprintf("/problem_with_error")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := errorProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Empty(t, m["status"])
			require.Empty(t, m["detail"])
			require.Empty(t, m["instance"])
		}
	}
}
