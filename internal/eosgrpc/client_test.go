package eosgrpc

import "testing"

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
