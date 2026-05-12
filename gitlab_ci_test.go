package main

import (
	"os"
	"strings"
	"testing"
)

func TestGitLabReleaseFetchDoesNotInstallConflictingCoreutils(t *testing.T) {
	data, err := os.ReadFile(".gitlab-ci.yml")
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "dnf install") && strings.Contains(line, "coreutils") {
			t.Fatalf("fetch job must not install coreutils because alma9-base already provides coreutils-single: %s", line)
		}
	}
}
