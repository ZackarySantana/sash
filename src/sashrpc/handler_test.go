package sashrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zackarysantana/sash/src/sashrpc"
)

type widget struct {
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

type calc struct{}

func (*calc) Ping(ctx context.Context) error { return nil }

func (*calc) Sum(ctx context.Context, a, b int) (int, error) { return a + b, nil }

func (*calc) EchoWidget(ctx context.Context, w widget) (widget, error) { return w, nil }

func TestHandlerSum(t *testing.T) {
	h, err := sashrpc.Handler(&calc{})
	if err != nil {
		t.Fatal(err)
	}
	body := map[string][]json.RawMessage{"args": {json.RawMessage("3"), json.RawMessage("4")}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/Sum", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		OK     bool `json:"ok"`
		Result int  `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Result != 7 {
		t.Fatalf("got %+v", out)
	}
}

func TestHandlerEchoStruct(t *testing.T) {
	h, err := sashrpc.Handler(&calc{})
	if err != nil {
		t.Fatal(err)
	}
	arg := widget{ID: "w1", Tags: []string{"a", "b"}}
	argRaw, err := json.Marshal(arg)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string][]json.RawMessage{"args": {json.RawMessage(argRaw)}}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/EchoWidget", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		OK     bool   `json:"ok"`
		Result widget `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Result.ID != "w1" || len(out.Result.Tags) != 2 {
		t.Fatalf("got %+v", out)
	}
}
