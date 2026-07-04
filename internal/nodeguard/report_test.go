package nodeguard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPReporterReturnsNon2xxStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	err := HTTPReporter{PanelURL: srv.URL, Client: srv.Client()}.Report(ReportPayload{Token: "secret-token", NodeID: 70, ActualHost: "bad.example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "status=422") || !strings.Contains(msg, "validation failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(msg, "secret-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}
