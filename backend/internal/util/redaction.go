package util

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

var sensitiveFields = map[string]bool{
	"password": true, "password_hash": true, "jwt": true, "token": true, "authorization": true,
	"private_key": true, "certificate_pem": true, "certificates_pem": true, "pem_redacted": true,
	"public_chain_pem": true, "input_snapshot": true,
}

func RedactValue(key string, value any) any {
	if sensitiveFields[strings.ToLower(key)] {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for nestedKey, nestedValue := range typed {
			result[nestedKey] = RedactValue(nestedKey, nestedValue)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = RedactValue(key, item)
		}
		return result
	default:
		return value
	}
}

func RedactedJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return "{}"
	}
	redacted := RedactValue("", decoded)
	output, err := json.Marshal(redacted)
	if err != nil {
		return "{}"
	}
	return string(output)
}

type redactingHandler struct{ next slog.Handler }

func NewRedactingLogger(next slog.Handler) *slog.Logger {
	return slog.New(&redactingHandler{next: next})
}
func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}
func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	copyRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool { copyRecord.AddAttrs(redactAttr(attr)); return true })
	return h.next.Handle(ctx, copyRecord)
}
func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		clean[i] = redactAttr(a)
	}
	return &redactingHandler{next: h.next.WithAttrs(clean)}
}
func (h *redactingHandler) WithGroup(name string) slog.Handler {
	return &redactingHandler{next: h.next.WithGroup(name)}
}
func redactAttr(attr slog.Attr) slog.Attr {
	if sensitiveFields[strings.ToLower(attr.Key)] {
		return slog.String(attr.Key, "[REDACTED]")
	}
	return attr
}
