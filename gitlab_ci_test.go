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

func TestGitLabPublishJobUsesStorageCIRunner(t *testing.T) {
	data, err := os.ReadFile(".gitlab-ci.yml")
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}
	if !strings.Contains(string(data), "docker_node") {
		t.Fatal("publish job should use docker_node, matching the Storage CI publishing pattern")
	}
}

func TestGitLabPublishMatchesEOSExporterSudoInvocation(t *testing.T) {
	data, err := os.ReadFile(".gitlab-ci.yml")
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}
	ci := string(data)
	if !strings.Contains(ci, "sudo -u stci -H ./gitlab-ci/publish_artifacts.sh") {
		t.Fatal("publish job should keep the plain eos_exporter-style sudo invocation")
	}
	if strings.Contains(ci, "sudo -u stci -H env") {
		t.Fatal("publish job should not depend on CI environment being passed through sudo")
	}
}

func TestGitLabPublishCanRecoverTagFromReleaseMetadata(t *testing.T) {
	data, err := os.ReadFile("gitlab-ci/publish_artifacts.sh")
	if err != nil {
		t.Fatalf("read publish script: %v", err)
	}
	script := string(data)
	for _, want := range []string{`"${SRC_DIR}/release.json"`, `"tag_name"`} {
		if !strings.Contains(script, want) {
			t.Fatalf("publish script must recover missing sudo-stripped tag from release metadata using %s", want)
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
