package eos

import "testing"

func TestParseVIDList(t *testing.T) {
	input := []byte(`
publicaccesslevel: => 1024
sudoer                 => uids()
tokensudo              => always
`)

	got := parseVIDList(input)
	want := []VIDRecord{
		{Key: "publicaccesslevel", Value: "1024"},
		{Key: "sudoer", Value: "uids()"},
		{Key: "tokensudo", Value: "always"},
	}

	if len(got) != len(want) {
		t.Fatalf("parseVIDList() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseVIDList()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParseVIDListSkipsPreambleAndPreservesBareKeys(t *testing.T) {
	input := []byte(`
* annotation
gateway rule
user:krb5:alice => vuid:1000
`)

	got := parseVIDList(input)
	want := []VIDRecord{
		{Key: "gateway rule", Value: ""},
		{Key: "user:krb5:alice", Value: "vuid:1000"},
	}

	if len(got) != len(want) {
		t.Fatalf("parseVIDList() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseVIDList()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
