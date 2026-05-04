package bindgen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/zackarysantana/sash/src/config"
)

var sseFieldKinds = map[string]string{
	"string": "string", "number": "number", "boolean": "boolean", "unknown": "unknown",
}

func validateSSEDeclarations(decls []config.SSEEventDecl) error {
	seen := map[string]struct{}{}
	for i, d := range decls {
		ev := strings.TrimSpace(d.Event)
		if ev == "" {
			return fmt.Errorf("bindings.sseEvents[%d]: event name is required", i)
		}
		if _, dup := seen[ev]; dup {
			return fmt.Errorf("bindings.sseEvents: duplicate event %q", ev)
		}
		seen[ev] = struct{}{}
		for fk, fv := range d.Fields {
			fk = strings.TrimSpace(fk)
			if fk == "" {
				return fmt.Errorf("bindings.sseEvents event %q: empty field name", ev)
			}
			norm := strings.ToLower(strings.TrimSpace(fv))
			if _, ok := sseFieldKinds[norm]; !ok {
				return fmt.Errorf("bindings.sseEvents event %q field %q: kind must be string, number, boolean, or unknown (got %q)", ev, fk, fv)
			}
		}
	}
	return nil
}

func appendSSEClientJS(b *bytes.Buffer) {
	fmt.Fprintf(b, "\nexport function createSashSSESession(opts = {}) {\n")
	fmt.Fprintf(b, "  const url = opts.url ?? eventsURL();\n")
	fmt.Fprintf(b, "  const es = new EventSource(url);\n")
	fmt.Fprintf(b, "  es.onerror = () => {};\n")
	fmt.Fprintf(b, "  return {\n")
	fmt.Fprintf(b, "    source: es,\n")
	fmt.Fprintf(b, "    listenJSON(eventName, handler) {\n")
	fmt.Fprintf(b, "      const fn = (ev) => {\n")
	fmt.Fprintf(b, "        let payload;\n")
	fmt.Fprintf(b, "        try {\n")
	fmt.Fprintf(b, "          payload = JSON.parse(ev.data);\n")
	fmt.Fprintf(b, "        } catch {\n")
	fmt.Fprintf(b, "          return;\n")
	fmt.Fprintf(b, "        }\n")
	fmt.Fprintf(b, "        handler(payload, ev);\n")
	fmt.Fprintf(b, "      };\n")
	fmt.Fprintf(b, "      es.addEventListener(eventName, fn);\n")
	fmt.Fprintf(b, "      return () => es.removeEventListener(eventName, fn);\n")
	fmt.Fprintf(b, "    },\n")
	fmt.Fprintf(b, "    close() {\n")
	fmt.Fprintf(b, "      es.close();\n")
	fmt.Fprintf(b, "    },\n")
	fmt.Fprintf(b, "  };\n")
	fmt.Fprintf(b, "}\n")
}

func appendSSEDTS(b *bytes.Buffer, decls []config.SSEEventDecl) {
	fmt.Fprintf(b, "\nexport interface SashSSESessionOpts {\n")
	fmt.Fprintf(b, "  url?: string;\n")
	fmt.Fprintf(b, "}\n\n")

	if len(decls) == 0 {
		fmt.Fprintf(b, "export interface SashSSESession {\n")
		fmt.Fprintf(b, "  readonly source: EventSource;\n")
		fmt.Fprintf(b, "  listenJSON<E extends string, T>(event: E, handler: (payload: T, ev: MessageEvent) => void): () => void;\n")
		fmt.Fprintf(b, "  close(): void;\n")
		fmt.Fprintf(b, "}\n\n")
		fmt.Fprintf(b, "export function createSashSSESession(opts?: SashSSESessionOpts): SashSSESession;\n")
		return
	}

	fmt.Fprintf(b, "export interface SashSSEPayloadMap {\n")
	events := make([]string, 0, len(decls))
	declByEvent := make(map[string]config.SSEEventDecl, len(decls))
	for _, d := range decls {
		ev := strings.TrimSpace(d.Event)
		events = append(events, ev)
		declByEvent[ev] = d
	}
	sort.Strings(events)
	for _, ev := range events {
		d := declByEvent[ev]
		fmt.Fprintf(b, "  %q: %s;\n", ev, ssePayloadTypeExpr(d.Fields))
	}
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "export type SashSSEEventName = keyof SashSSEPayloadMap;\n\n")

	fmt.Fprintf(b, "export interface SashSSESession {\n")
	fmt.Fprintf(b, "  readonly source: EventSource;\n")
	fmt.Fprintf(b, "  listenJSON<N extends keyof SashSSEPayloadMap>(event: N, handler: (payload: SashSSEPayloadMap[N], ev: MessageEvent) => void): () => void;\n")
	fmt.Fprintf(b, "  close(): void;\n")
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "export function createSashSSESession(opts?: SashSSESessionOpts): SashSSESession;\n")
}

func ssePayloadTypeExpr(fields map[string]string) string {
	if len(fields) == 0 {
		return "unknown"
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, k := range keys {
		norm := strings.ToLower(strings.TrimSpace(fields[k]))
		tsKind := sseFieldKinds[norm]
		sb.WriteString(tsRecordPropLine(strings.TrimSpace(k), tsKind))
	}
	sb.WriteString("  }")
	return sb.String()
}

func tsRecordPropLine(name, tsType string) string {
	if isASCIIJSIdent(name) {
		return fmt.Sprintf("    %s: %s;\n", name, tsType)
	}
	return fmt.Sprintf("    %q: %s;\n", name, tsType)
}

func isASCIIJSIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		r := s[i]
		if i == 0 {
			if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				return false
			}
			continue
		}
		if r != '_' && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}
