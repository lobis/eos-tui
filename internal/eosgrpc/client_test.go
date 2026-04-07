package eosgrpc

import (
	"encoding/json"
	"testing"
)

func TestParseStatusHealth(t *testing.T) {
	input := "instance: eosdev\n          health:     OK       \n"
	if got := parseStatusHealth(input); got != "OK" {
		t.Fatalf("expected OK, got %q", got)
	}
}

func TestParseLabeledValues(t *testing.T) {
	input := `
ALL      Files                            78 [booted] (0s)
ALL      Directories                      19
ALL      current file id                  7661
ALL      memory resident                  586.30 MB
`
	values := parseLabeledValues(input)

	if got := parseUint(values["Files"]); got != 78 {
		t.Fatalf("expected files=78, got %d", got)
	}
	if got := parseUint(values["Directories"]); got != 19 {
		t.Fatalf("expected directories=19, got %d", got)
	}
	if got := parseUint(values["current file id"]); got != 7661 {
		t.Fatalf("expected current file id=7661, got %d", got)
	}
	if got := parseHumanBytes(values["memory resident"]); got == 0 {
		t.Fatalf("expected parsed memory resident bytes, got 0")
	}
}

func TestEntryFromCLIContainer(t *testing.T) {
	entry := entryFromCLI(cliFileInfo{
		Name:           "eos",
		Path:           "/eos/",
		ID:             2,
		PID:            1,
		Inode:          2,
		UID:            0,
		GID:            0,
		Mode:           16893,
		TreeFiles:      78,
		TreeContainers: 17,
		TreeSize:       4907360263,
	})

	if entry.Kind != EntryKindContainer {
		t.Fatalf("expected container kind, got %q", entry.Kind)
	}
	if entry.Path != "/eos" {
		t.Fatalf("expected cleaned path /eos, got %q", entry.Path)
	}
	if entry.Files != 78 || entry.Containers != 17 {
		t.Fatalf("unexpected tree counts: files=%d containers=%d", entry.Files, entry.Containers)
	}
}

func TestEntryFromCLIFile(t *testing.T) {
	entry := entryFromCLI(cliFileInfo{
		Name:      "hola",
		Path:      "/eos/dev/test/hola",
		ID:        14,
		PID:       17,
		Inode:     9223372036854775822,
		UID:       0,
		GID:       0,
		Mode:      493,
		Size:      12,
		Locations: []cliLocation{{FSID: 3}},
	})

	if entry.Kind != EntryKindFile {
		t.Fatalf("expected file kind, got %q", entry.Kind)
	}
	if entry.Size != 12 || entry.Locations != 1 {
		t.Fatalf("unexpected file metadata: size=%d locations=%d", entry.Size, entry.Locations)
	}
}

func TestParseSpaceStatus(t *testing.T) {
	input := `
groupbalancer.threshold          := 5
groupmod                         := 24
lru                              := on
tgc.totalbytes                   := 1000000000000000000
`
	records := parseSpaceStatus([]byte(input))

	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}

	if records[0].Key != "groupbalancer.threshold" || records[0].Value != "5" {
		t.Fatalf("unexpected record 0: %+v", records[0])
	}
	if records[3].Key != "tgc.totalbytes" || records[3].Value != "1000000000000000000" {
		t.Fatalf("unexpected record 3: %+v", records[3])
	}
}

func TestNamespaceStatsJSONConflict(t *testing.T) {
	// Simulated JSON with conflicting types for 'files' and 'directories'
	input := `
{
	"result": [
		{
			"ns": {
				"total": {
					"files": 78,
					"directories": 19
				}
			}
		},
		{
			"ns": {
				"total": {
					"files": {
						"changelog": { "size": 0 }
					}
				}
			}
		}
	],
	"retc": "0"
}
`
	var payload struct {
		Result []struct {
			NS struct {
				Total struct {
					Files       any `json:"files"`
					Directories any `json:"directories"`
				} `json:"total"`
			} `json:"ns"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(payload.Result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(payload.Result))
	}

	if val := toUint64(payload.Result[0].NS.Total.Files); val != 78 {
		t.Fatalf("expected result[0] files=78, got %v", val)
	}

	if val := toUint64(payload.Result[1].NS.Total.Files); val != 0 {
		t.Fatalf("expected result[1] files=0 (ignored object), got %v", val)
	}
}

func TestToUint64(t *testing.T) {
	tests := []struct {
		input any
		want  uint64
	}{
		{float64(78), 78},
		{uint64(100), 100},
		{int64(50), 50},
		{int(10), 10},
		{"not a number", 0},
		{map[string]any{"foo": "bar"}, 0},
	}

	for _, tt := range tests {
		if got := toUint64(tt.input); got != tt.want {
			t.Errorf("toUint64(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
