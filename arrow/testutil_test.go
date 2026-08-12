package arrow

import (
	"io"
	"net/http"
	"net/http/httptest"
)

func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient("test-app-id", "test-app-secret")
	client.Config.BaseURL = server.URL
	client.Config.Token = "test-token"
	return client, server
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
