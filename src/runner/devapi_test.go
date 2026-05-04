package runner

import (
	"net/http"
	"testing"
)

func TestStartDevAPIServerPing(t *testing.T) {
	baseURL, stop, err := startDevAPIServer("127.0.0.1:0", func(mux *http.ServeMux) {
		mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	res, err := http.Get(baseURL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

func TestRunDevAPIMountRequiresListenAddr(t *testing.T) {
	t.Setenv("SASH_API_LISTEN", "")
	err := Run(Options{
		DevURL:      "http://127.0.0.1:59997",
		DevAPIAddr:  "",
		DevAPIMount: func(mux *http.ServeMux) {},
		Runner:      &StubRunner{},
		OpenBrowser: false,
	})
	if err == nil {
		t.Fatal("expected error when DevAPIMount is set but listen addr is missing")
	}
}
