// Tests for the dashboard package.
package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStart(t *testing.T) {
	cases := []struct {
		method         string
		pattern        string
		wantCode       int
		wantInResponse string
		auth           bool
	}{
		{"GET", "/", 200, "g-signin", true},
		{"GET", "/", 200, "WebProber", false},
	}
	for i, tt := range cases {
		*authDisabled = !tt.auth
		router := Start()

		req, err := http.NewRequest(tt.method, tt.pattern, nil)
		if err != nil {
			t.Fatalf("[%d] failed to create %s %s request: %v\n", i, tt.method, tt.pattern, err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != tt.wantCode {
			t.Fatalf("[%d] want HTTP response %d for %s %s, got %d\n", i, tt.wantCode, tt.method, tt.pattern, w.Code)
		}
		if !strings.Contains(w.Body.String(), tt.wantInResponse) {
			t.Fatalf("[%d] want %q in response for %s %s, didn't get it: \n%s\n", i, tt.wantInResponse, tt.method, tt.pattern, w.Body.String())
		}
	}
}
