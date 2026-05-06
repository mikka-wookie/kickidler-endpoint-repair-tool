package app

import (
	"encoding/json"
	"reflect"
	"strings"

	"kigrepair/internal/safety"
)

func SafeRequestSummary(req any) map[string]any {
	data, err := json.Marshal(req)
	if err != nil {
		return map[string]any{"summary": safety.RedactString(err.Error())}
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		return map[string]any{"summary": safety.RedactString(string(data))}
	}
	if inviteValue(req) != "" {
		summary["invite"] = safety.RedactedValue
	}
	return summary
}

func RedactProgressEvent(event ProgressEvent) ProgressEvent {
	event.Message = safety.RedactString(event.Message)
	event.ResultFile = safety.RedactString(event.ResultFile)
	event.FailureCategory = safety.RedactString(event.FailureCategory)
	return event
}

type RedactingProgressSink struct {
	Next ProgressSink
}

func (s RedactingProgressSink) OnProgress(event ProgressEvent) {
	if s.Next == nil {
		return
	}
	s.Next.OnProgress(RedactProgressEvent(event))
}

func inviteValue(value any) string {
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}
	if field := v.FieldByName("InviteValue"); field.IsValid() && field.Kind() == reflect.String {
		return strings.TrimSpace(field.String())
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Struct {
			if value := inviteValue(field.Interface()); value != "" {
				return value
			}
		}
	}
	return ""
}
