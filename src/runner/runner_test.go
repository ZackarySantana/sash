package runner_test

import (
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/zackarysantana/sash/src/runner"
)

func TestRunAssetsRecordsURLWithStub(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>x</title>")},
	}
	stub := &runner.StubRunner{}
	err := runner.Run(runner.Options{
		Assets: http.FS(fs),
		Runner: stub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stub.LastURL == "" || stub.LastURL[:4] != "http" {
		t.Fatalf("expected loopback http URL, got %q", stub.LastURL)
	}
}
