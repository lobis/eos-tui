package eos

import (
	"encoding/json"
	"testing"
)

func TestFlexibleStringUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain string", `"hello"`, "hello"},
		{"empty string", `""`, ""},
		{"null", `null`, ""},
		{"integer", `83737789`, "83737789"},
		{"float", `3.14`, "3.14"},
		{"negative", `-42`, "-42"},
		{"bool true", `true`, "true"},
		{"bool false", `false`, "false"},
		{"whitespace integer", `   12345  `, "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s flexibleString
			if err := s.UnmarshalJSON([]byte(tt.in)); err != nil {
				t.Fatalf("UnmarshalJSON(%q) error: %v", tt.in, err)
			}
			if string(s) != tt.want {
				t.Fatalf("UnmarshalJSON(%q) = %q, want %q", tt.in, string(s), tt.want)
			}
		})
	}
}

func TestFlexibleStringInStruct(t *testing.T) {
	type wrapper struct {
		Geotag flexibleString `json:"geotag"`
	}
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"string", `{"geotag":"eu/cern/0123"}`, "eu/cern/0123"},
		{"number", `{"geotag":83737789}`, "83737789"},
		{"missing", `{}`, ""},
		{"null", `{"geotag":null}`, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var w wrapper
			if err := json.Unmarshal([]byte(tt.raw), &w); err != nil {
				t.Fatalf("Unmarshal(%q) error: %v", tt.raw, err)
			}
			if string(w.Geotag) != tt.want {
				t.Fatalf("Geotag = %q, want %q", string(w.Geotag), tt.want)
			}
		})
	}
}
