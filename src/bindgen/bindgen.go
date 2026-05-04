package bindgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/zackarysantana/sash/src/config"
	"golang.org/x/tools/go/packages"
)

const generatedGoBindingsPkg = "sashbindings"

func Generate(cfg *config.Bindings, projectDir, moduleDir string) error {
	if cfg == nil {
		return errors.New("bindings config is nil")
	}
	if strings.TrimSpace(cfg.APIImportPath) == "" {
		return errors.New("bindings.apiImportPath is required")
	}
	if strings.TrimSpace(cfg.Type) == "" {
		return errors.New("bindings.type is required")
	}
	if strings.TrimSpace(cfg.GoBindingsDir) == "" || strings.TrimSpace(cfg.TSBindingsDir) == "" {
		return errors.New("bindings.goBindingsDir and bindings.tsBindingsDir are required")
	}

	mount := strings.TrimSpace(cfg.MountPath)
	if mount == "" {
		return errors.New("bindings.mountPath is required")
	}
	if !strings.HasPrefix(mount, "/") {
		mount = "/" + mount
	}
	mount = strings.TrimSuffix(mount, "/")

	devAddr := strings.TrimSpace(cfg.DevListenAddr)
	if devAddr == "" {
		return errors.New("bindings.devListenAddr is required")
	}
	devAPIBase := strings.TrimSpace(cfg.DevAPIBaseURL)
	if devAPIBase == "" {
		return errors.New("bindings.devAPIBaseURL is required")
	}
	devAPIBase = strings.TrimSuffix(devAPIBase, "/")

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedImports | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule | packages.NeedName,
		Dir:  moduleDir,
	}, cfg.APIImportPath)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("loading package %s: no packages returned", cfg.APIImportPath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var b strings.Builder
		for _, e := range pkg.Errors {
			fmt.Fprintf(&b, "%v\n", e)
		}
		return fmt.Errorf("loading package %s:\n%s", cfg.APIImportPath, b.String())
	}

	methods, err := loadMethods(pkg, cfg.Type)
	if err != nil {
		return err
	}
	if err := validateSSEDeclarations(cfg.SSEEvents); err != nil {
		return err
	}

	goDir := filepath.Join(projectDir, filepath.Clean(cfg.GoBindingsDir))
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		return err
	}
	tsDir := filepath.Join(projectDir, filepath.Clean(cfg.TSBindingsDir))
	if err := os.MkdirAll(tsDir, 0o755); err != nil {
		return err
	}

	apiPkgIdent := pkg.Name
	if apiPkgIdent == generatedGoBindingsPkg {
		return fmt.Errorf("API package name %q clashes with generated bindings package %q; rename the API package", apiPkgIdent, generatedGoBindingsPkg)
	}

	goSrc, err := renderGo(cfg.APIImportPath, apiPkgIdent, cfg.Type, mount, devAddr)
	if err != nil {
		return err
	}
	goFmt, err := format.Source(goSrc)
	if err != nil {
		return fmt.Errorf("format generated Go: %w\n%s", err, string(goSrc))
	}
	goPath := filepath.Join(goDir, "bindings_gen.go")
	if err := os.WriteFile(goPath, goFmt, 0o644); err != nil {
		return err
	}

	tsJS := renderClientJS(cfg.Type, methods, mount, devAPIBase)
	if err := os.WriteFile(filepath.Join(tsDir, "index.js"), []byte(tsJS), 0o644); err != nil {
		return err
	}
	tsDTS := renderClientDTS(cfg.Type, methods, cfg.SSEEvents)
	if err := os.WriteFile(filepath.Join(tsDir, "index.d.ts"), []byte(tsDTS), 0o644); err != nil {
		return err
	}

	return nil
}

func loadMethods(pkg *packages.Package, typeName string) ([]signatureMethod, error) {
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("unknown type %s in package %s", typeName, pkg.PkgPath)
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s is not a named type", typeName)
	}
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("%s must be a struct type", typeName)
	}

	ptr := types.NewPointer(named)
	ms := types.NewMethodSet(ptr)

	var out []signatureMethod
	for i := 0; i < ms.Len(); i++ {
		sel := ms.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok {
			continue
		}
		sm, err := parseSignature(fn.Name(), sig)
		if err != nil {
			return nil, fmt.Errorf("method %s: %w", fn.Name(), err)
		}
		for pi, pt := range sm.ParamTypes {
			if err := validateBindingWireType(pt, fmt.Sprintf("method %s parameter %d", fn.Name(), pi)); err != nil {
				return nil, fmt.Errorf("method %s: %w", fn.Name(), err)
			}
		}
		if sm.ResultKind == "valueError" {
			if err := validateBindingWireType(sm.ResultType, fmt.Sprintf("method %s result", fn.Name())); err != nil {
				return nil, fmt.Errorf("method %s: %w", fn.Name(), err)
			}
		}
		out = append(out, sm)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no exported methods with supported signatures on *%s", typeName)
	}
	return out, nil
}

type signatureMethod struct {
	Name       string
	HasCtx     bool
	ParamTypes []types.Type
	ResultKind string
	ResultType types.Type
}

func parseSignature(name string, sig *types.Signature) (signatureMethod, error) {
	params := sig.Params()
	results := sig.Results()
	nParams := params.Len()
	nResults := results.Len()
	if nParams < 1 {
		return signatureMethod{}, errors.New("missing receiver")
	}

	hasCtx := false
	firstArg := 1
	if nParams >= 2 && isContextType(params.At(1).Type()) {
		hasCtx = true
		firstArg = 2
	}

	var paramTypes []types.Type
	for i := firstArg; i < nParams; i++ {
		paramTypes = append(paramTypes, params.At(i).Type())
	}

	errObj := types.Universe.Lookup("error")
	errTN, ok := errObj.(*types.TypeName)
	if !ok {
		return signatureMethod{}, fmt.Errorf("internal: unexpected builtin error object %T", errObj)
	}
	errorIface := errTN.Type().Underlying().(*types.Interface)

	switch nResults {
	case 1:
		if !types.Implements(results.At(0).Type(), errorIface) {
			return signatureMethod{}, errors.New("must return error or (T, error)")
		}
		return signatureMethod{Name: name, HasCtx: hasCtx, ParamTypes: paramTypes, ResultKind: "error"}, nil
	case 2:
		if !types.Implements(results.At(1).Type(), errorIface) {
			return signatureMethod{}, errors.New("second result must be error")
		}
		return signatureMethod{
			Name:       name,
			HasCtx:     hasCtx,
			ParamTypes: paramTypes,
			ResultKind: "valueError",
			ResultType: results.At(0).Type(),
		}, nil
	default:
		return signatureMethod{}, errors.New("must return (error) or (T, error)")
	}
}

func isContextType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func validateBindingWireType(t types.Type, ctx string) error {
	if t == nil {
		return nil
	}
	switch tt := t.(type) {
	case *types.Alias:
		return validateBindingWireType(tt.Rhs(), ctx)
	case *types.Named:
		return validateBindingWireType(tt.Underlying(), ctx)
	case *types.Pointer:
		return validateBindingWireType(tt.Elem(), ctx)
	case *types.Slice:
		return validateBindingWireType(tt.Elem(), ctx+" element")
	case *types.Array:
		return validateBindingWireType(tt.Elem(), ctx+" element")
	case *types.Map:
		if !isJSONObjectMapKey(tt.Key()) {
			return fmt.Errorf("%s: JSON maps need string keys (got %s)", ctx, tt.Key().String())
		}
		return validateBindingWireType(tt.Elem(), ctx+" value")
	case *types.Struct:
		return nil
	case *types.Interface:
		if tt.NumMethods() > 0 {
			return fmt.Errorf("%s: non-empty interfaces are not JSON-RPC compatible", ctx)
		}
		return nil
	case *types.Signature:
		return fmt.Errorf("%s: func types are not JSON-RPC compatible", ctx)
	case *types.Chan:
		return fmt.Errorf("%s: channel types are not JSON-RPC compatible", ctx)
	case *types.Basic:
		switch tt.Kind() {
		case types.Complex64, types.Complex128, types.UnsafePointer:
			return fmt.Errorf("%s: type %s is not JSON-RPC compatible", ctx, t.String())
		}
		return nil
	case *types.Tuple:
		return fmt.Errorf("%s: tuple types are not JSON-RPC compatible", ctx)
	default:
		return fmt.Errorf("%s: unsupported type %s for JSON-RPC", ctx, t.String())
	}
}

func isJSONObjectMapKey(key types.Type) bool {
	switch k := key.(type) {
	case *types.Alias:
		return isJSONObjectMapKey(k.Rhs())
	case *types.Named:
		return isJSONObjectMapKey(k.Underlying())
	case *types.Basic:
		return k.Kind() == types.String || k.Kind() == types.UntypedString
	default:
		return false
	}
}

func renderGo(apiImportPath, apiPkgIdent, structName, mount, devAddr string) ([]byte, error) {
	var b bytes.Buffer
	recv := "*" + apiPkgIdent + "." + structName
	fmt.Fprintf(&b, "// Code generated by \"sash bind\"; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", generatedGoBindingsPkg)
	fmt.Fprintf(&b, "import (\n")
	fmt.Fprintf(&b, "\t\"net/http\"\n\n")
	fmt.Fprintf(&b, "\t%s %q\n\n", apiPkgIdent, apiImportPath)
	fmt.Fprintf(&b, "\t\"github.com/zackarysantana/sash/src/sashrpc\"\n")
	fmt.Fprintf(&b, ")\n\n")

	fmt.Fprintf(&b, "const EmbeddedMountPath = %q\n\n", mount)
	fmt.Fprintf(&b, "const DevListenAddr = %q\n\n", devAddr)

	fmt.Fprintf(&b, "func RPCHandler(recv %s) http.Handler {\n", recv)
	fmt.Fprintf(&b, "\th, err := sashrpc.Handler(recv)\n")
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\tpanic(err)\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\treturn sashrpc.WithCORS(h)\n")
	fmt.Fprintf(&b, "}\n\n")

	fmt.Fprintf(&b, "func MountEmbedded(mux *http.ServeMux, recv %s) {\n", recv)
	fmt.Fprintf(&b, "\tmux.Handle(EmbeddedMountPath+\"/API/\", http.StripPrefix(EmbeddedMountPath+\"/API\", RPCHandler(recv)))\n")
	fmt.Fprintf(&b, "}\n\n")

	fmt.Fprintf(&b, "func MountDevRoutes(mux *http.ServeMux, recv %s) {\n", recv)
	fmt.Fprintf(&b, "\tmux.Handle(\"/API/\", http.StripPrefix(\"/API\", RPCHandler(recv)))\n")
	fmt.Fprintf(&b, "}\n")

	return b.Bytes(), nil
}
