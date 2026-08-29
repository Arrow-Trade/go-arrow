package arrow

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestNewClientSetsDefaultTimeout(t *testing.T) {
	client := NewClient("app", "secret")
	if client.HTTPTimeout() != DefaultHTTPTimeout {
		t.Fatalf("HTTPTimeout = %v, want %v", client.HTTPTimeout(), DefaultHTTPTimeout)
	}
	if client.HTTPClient.ReadTimeout != DefaultHTTPTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", client.HTTPClient.ReadTimeout, DefaultHTTPTimeout)
	}
	if client.HTTPClient.WriteTimeout != DefaultHTTPTimeout {
		t.Fatalf("WriteTimeout = %v, want %v", client.HTTPClient.WriteTimeout, DefaultHTTPTimeout)
	}
}

func TestNewClientWithTimeoutAndSetter(t *testing.T) {
	client := NewClientWithTimeout("app", "secret", 30*time.Second)
	if client.HTTPTimeout() != 30*time.Second {
		t.Fatalf("HTTPTimeout = %v, want 30s", client.HTTPTimeout())
	}
	client.SetHTTPTimeout(5 * time.Second)
	if client.HTTPTimeout() != 5*time.Second {
		t.Fatalf("after SetHTTPTimeout: %v, want 5s", client.HTTPTimeout())
	}
	if client.HTTPClient.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %v, want 5s", client.HTTPClient.ReadTimeout)
	}
	client.SetHTTPTimeout(0)
	if client.HTTPTimeout() != 5*time.Second {
		t.Fatalf("zero SetHTTPTimeout should be ignored, got %v", client.HTTPTimeout())
	}
}

func TestRESTCallTimesOutWhenServerStalls(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	})
	defer server.Close()
	client.SetHTTPTimeout(200 * time.Millisecond)

	start := time.Now()
	_, err := client.request("/info/holidays", "GET", nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Fatalf("blocked for %v, want return within ~200ms", elapsed)
	}
	if err != fasthttp.ErrTimeout && !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("err = %v, want a timeout", err)
	}
}
