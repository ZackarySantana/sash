package sashrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"
)

func Handler(recv interface{}) (http.Handler, error) {
	if recv == nil {
		return nil, errors.New("sashrpc: recv is nil")
	}
	v := reflect.ValueOf(recv)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil, errors.New("sashrpc: recv must be non-nil pointer to struct")
	}
	t := v.Type()
	methods := map[string]reflect.Method{}
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if !m.IsExported() {
			continue
		}
		if err := validateMethod(m); err != nil {
			return nil, fmt.Errorf("sashrpc: method %s: %w", m.Name, err)
		}
		methods[m.Name] = m
	}
	if len(methods) == 0 {
		return nil, errors.New("sashrpc: no callable exported methods on recv")
	}
	return &dispatch{recv: v, methods: methods}, nil
}

func validateMethod(m reflect.Method) error {
	mt := m.Func.Type()
	if mt.NumIn() < 1 {
		return errors.New("invalid arity")
	}
	errType := reflect.TypeOf((*error)(nil)).Elem()
	switch mt.NumOut() {
	case 1:
		if !mt.Out(0).Implements(errType) {
			return errors.New("must return only error or (T, error)")
		}
	case 2:
		if mt.Out(1).Implements(errType) == false {
			return errors.New("second result must be error")
		}
	default:
		return errors.New("must return exactly one error or (T, error)")
	}
	return nil
}

type dispatch struct {
	recv    reflect.Value
	methods map[string]reflect.Method
}

type rpcRequest struct {
	Args []json.RawMessage `json:"args"`
}

func (d *dispatch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	base := strings.Trim(path.Clean("/"+r.URL.Path), "/")
	if base == "" {
		http.NotFound(w, r)
		return
	}
	methodName := strings.Split(base, "/")[0]
	m, ok := d.methods[methodName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	var body rpcRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	mt := m.Func.Type()
	hasCtx, argStart := ctxArgOffset(mt)
	numArgs := mt.NumIn() - argStart
	if len(body.Args) != numArgs {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("expected %d args, got %d", numArgs, len(body.Args))})
		return
	}

	in := make([]reflect.Value, mt.NumIn())
	in[0] = d.recv
	j := 1
	if hasCtx {
		in[j] = reflect.ValueOf(r.Context())
		j++
	}
	for k := 0; k < numArgs; k++ {
		argT := mt.In(j + k)
		ptr := reflect.New(argT)
		raw := body.Args[k]
		if len(raw) == 0 {
			raw = []byte("null")
		}
		if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("arg %d: %v", k, err)})
			return
		}
		in[j+k] = ptr.Elem()
	}

	out := m.Func.Call(in)

	switch mt.NumOut() {
	case 1:
		errVal := out[0]
		if !errVal.IsNil() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": errVal.Interface().(error).Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case 2:
		errVal := out[1]
		if !errVal.IsNil() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": errVal.Interface().(error).Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "result": out[0].Interface()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unsupported arity"})
	}
}

func ctxArgOffset(mt reflect.Type) (hasCtx bool, argStart int) {
	argStart = 1 // reflect Method.Type includes recv at In(0)
	if mt.NumIn() < 2 {
		return false, argStart
	}
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if mt.In(1) == ctxType {
		return true, 2
	}
	return false, argStart
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
