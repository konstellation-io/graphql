package graphql

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoJSON(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		_, err = io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	require.NoError(t, err)
	require.Equal(t, calls, 1) // calls
	require.Equal(t, responseData["something"], "yes")
}

func TestDoJSONServerError(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusInternalServerError)
		_, err = io.WriteString(w, `Internal Server Error`)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)

	require.Equal(t, calls, 1) // calls
	require.Equal(t, err.Error(), "graphql: server returned a non-200 status code: 500")
}

func TestDoJSONBadRequestErr(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusBadRequest)
		_, err = io.WriteString(w, `{
			"errors": [{
				"message": "miscellaneous message as to why the the request was bad"
			}]
		}`)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)

	require.Equal(t, calls, 1) // calls
	require.Equal(t, err.Error(), "graphql: miscellaneous message as to why the the request was bad")
}

func TestQueryJSON(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, string(b), `{"query":"query {}","variables":{"username":"matryer"}}`+"\n")
		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	require.True(t, req != nil)
	require.Equal(t, req.vars["username"], "matryer")

	var resp struct {
		Value string
	}

	err := client.Run(ctx, req, &resp)

	require.NoError(t, err)
	require.Equal(t, calls, 1)

	require.Equal(t, resp.Value, "some data")
}

func TestHeader(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Header.Get("X-Custom-Header"), "123")

		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Header.Set("X-Custom-Header", "123")

	var resp struct {
		Value string
	}

	err := client.Run(ctx, req, &resp)
	require.NoError(t, err)
	require.Equal(t, calls, 1)

	require.Equal(t, resp.Value, "some data")
}
