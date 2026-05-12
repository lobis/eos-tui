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

func TestGitHubReleaseChecksumsExcludeChecksumFile(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	if !strings.Contains(string(data), "! -name SHA256SUMS.txt") {
		t.Fatal("release workflow must exclude SHA256SUMS.txt when generating SHA256SUMS.txt")
	}
}

func TestGitLabReleaseFetchIgnoresChecksumFileSelfEntry(t *testing.T) {
	data, err := os.ReadFile("gitlab-ci/fetch_github_release.sh")
	if err != nil {
		t.Fatalf("read fetch script: %v", err)
	}
	if !strings.Contains(string(data), "SHA256SUMS\\.txt") || !strings.Contains(string(data), "sha256sum -c -") {
		t.Fatal("fetch script must filter SHA256SUMS.txt self-entry before checksum verification")
	}
}

func TestGitLabReleaseFetchDoesNotExpandEmptyAuthArray(t *testing.T) {
	data, err := os.ReadFile("gitlab-ci/fetch_github_release.sh")
	if err != nil {
		t.Fatalf("read fetch script: %v", err)
	}
	if strings.Contains(string(data), "CURL_AUTH[@]") {
		t.Fatal("fetch script must not expand an empty CURL_AUTH array under set -u")
	}
}

func TestGitLabReleaseFetchAvoidsMapfileForLocalBashCompatibility(t *testing.T) {
	data, err := os.ReadFile("gitlab-ci/fetch_github_release.sh")
	if err != nil {
		t.Fatalf("read fetch script: %v", err)
	}
	if strings.Contains(string(data), "mapfile") {
		t.Fatal("fetch script must avoid mapfile so it can be reproduced with older local Bash versions")
	}
}
