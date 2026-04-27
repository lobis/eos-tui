package eos

import "testing"

// Regression for https://github.com/cern-eos/eos-tui — EOS sometimes emits
// stat.geotag as a JSON number rather than a string. parseFileSystemsJSON
// must accept both.
func TestParseFileSystemsJSONAcceptsNumericGeotag(t *testing.T) {
	output := []byte(`{
  "errormsg": "",
  "result": [
    {
      "configstatus": "rw",
      "host": "st-096-o2-18844e.cern.ch",
      "id": 14921,
      "path": "/data/fst.1/14921",
      "schedgroup": "default.0",
      "stat": {
        "active": "online",
        "boot": "booted",
        "geotag": 83737789,
        "health": {"status": "ok"},
        "disk": {"bw": 0, "iops": 0, "readratemb": 0, "writeratemb": 0},
        "statfs": {"capacity": 100, "freebytes": 50, "usedbytes": 50},
        "usedfiles": 7
      }
    },
    {
      "configstatus": "rw",
      "host": "st-096-o2-185bad.cern.ch",
      "id": 15168,
      "path": "/data/fst.1/15168",
      "schedgroup": "default.0",
      "stat": {
        "active": "online",
        "boot": "booted",
        "geotag": "eu/cern/0123",
        "health": {"status": "ok"},
        "disk": {"bw": 0, "iops": 0, "readratemb": 0, "writeratemb": 0},
        "statfs": {"capacity": 100, "freebytes": 50, "usedbytes": 50},
        "usedfiles": 0
      }
    }
  ]
}`)

	records, err := parseFileSystemsJSON(output)
	if err != nil {
		t.Fatalf("parseFileSystemsJSON: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	// Sort is by ID ascending, so 14921 first.
	if records[0].ID != 14921 {
		t.Fatalf("records[0].ID = %d, want 14921", records[0].ID)
	}
	if records[0].Geotag != "83737789" {
		t.Fatalf("numeric geotag = %q, want \"83737789\"", records[0].Geotag)
	}
	if records[1].Geotag != "eu/cern/0123" {
		t.Fatalf("string geotag = %q, want \"eu/cern/0123\"", records[1].Geotag)
	}
}
