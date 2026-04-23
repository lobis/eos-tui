package eos

import "testing"

func TestParseAccessList(t *testing.T) {
	output := []byte(`
user.banned=eosnobody
user.allowed=lobisapa
group.allowed=root
redirect=host-a:1094
`)

	got := parseAccessList(output)
	want := []AccessRecord{
		{Category: "user", Rule: "banned", Value: "eosnobody", RawKey: "user.banned"},
		{Category: "user", Rule: "allowed", Value: "lobisapa", RawKey: "user.allowed"},
		{Category: "group", Rule: "allowed", Value: "root", RawKey: "group.allowed"},
		{Category: "redirect", Rule: "value", Value: "host-a:1094", RawKey: "redirect"},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d records, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("record %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestAccessRuleArgs(t *testing.T) {
	got, err := accessRuleArgs("ban", "user", "lobisapa")
	if err != nil {
		t.Fatalf("accessRuleArgs returned error: %v", err)
	}

	want := []string{"eos", "access", "ban", "user", "lobisapa"}
	if len(got) != len(want) {
		t.Fatalf("arg count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAccessStallArgs(t *testing.T) {
	got, err := accessStallArgs(300)
	if err != nil {
		t.Fatalf("accessStallArgs returned error: %v", err)
	}

	want := []string{"eos", "access", "set", "stall", "300"}
	if len(got) != len(want) {
		t.Fatalf("arg count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}
