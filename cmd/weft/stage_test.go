package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The published site must keep every real release so WEFT_VERSION pins keep working,
// and must never carry a test build.
func TestStageSiteKeepsReleasesDropsTestBuilds(t *testing.T) {
	skipWindows(t)
	site := t.TempDir()
	for _, f := range []string{"index.html", "install"} {
		os.WriteFile(filepath.Join(site, f), []byte(f), 0o644)
	}
	for _, v := range []string{"0.0.1", "0.0.2", "0.0.0-test", "0.1.0-alpha.1"} {
		dir := filepath.Join(site, "releases", v)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(v), 0o644)
	}
	out := t.TempDir()
	cmd := exec.Command("sh", "../../scripts/stage-site.sh", site, out, "0.0.2")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	var got []string
	filepath.Walk(out, func(p string, fi os.FileInfo, _ error) error {
		if !fi.IsDir() {
			rel, _ := filepath.Rel(out, p)
			got = append(got, rel)
		}
		return nil
	})
	sort.Strings(got)
	want := []string{
		"index.html", "install",
		"releases/0.0.1/SHA256SUMS", "releases/0.0.2/SHA256SUMS",
		"releases/0.1.0-alpha.1/SHA256SUMS", "releases/latest",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("staged %v\nwant   %v", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "releases/latest")); string(b) != "0.0.2\n" {
		t.Fatalf("latest %q", b)
	}
}

// fakeSite writes a site/ like the repository's: the installer, the given
// releases (each with a SHA256SUMS) and, unless latest is "", releases/latest.
func fakeSite(t *testing.T, latest string, versions ...string) string {
	t.Helper()
	site := t.TempDir()
	os.WriteFile(filepath.Join(site, "install"), []byte("#!/bin/sh\n"), 0o644)
	for _, v := range versions {
		dir := filepath.Join(site, "releases", v)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(v), 0o644)
	}
	if latest != "" {
		os.WriteFile(filepath.Join(site, "releases", "latest"), []byte(latest+"\n"), 0o644)
	}
	return site
}

func files(t *testing.T, dir string) string {
	t.Helper()
	var got []string
	filepath.Walk(dir, func(p string, fi os.FileInfo, _ error) error {
		if fi != nil && !fi.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			got = append(got, rel)
		}
		return nil
	})
	sort.Strings(got)
	return strings.Join(got, ",")
}

func stageReleases(args ...string) (string, error) {
	b, err := exec.Command("sh", append([]string{"../../scripts/stage-releases.sh"}, args...)...).CombinedOutput()
	return string(b), err
}

// The Weft-built site stages the installer and every real release into its
// public/ beside the files already there, and can do it again.
func TestStageReleasesIntoSitePublic(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.1", "0.0.2", "0.0.0-test")
	public := t.TempDir()
	os.WriteFile(filepath.Join(public, "favicon.svg"), []byte("<svg/>"), 0o644)
	for range 2 {
		if out, err := stageReleases(site, public); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	want := "favicon.svg,install,releases/0.0.1/SHA256SUMS,releases/0.0.2/SHA256SUMS,releases/latest"
	if got := files(t, public); got != want {
		t.Fatalf("staged %s\nwant   %s", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(public, "releases/latest")); string(b) != "0.0.2\n" {
		t.Fatalf("latest %q", b)
	}
	// A version given names latest.
	if out, err := stageReleases(site, public, "0.0.1"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(public, "releases/latest")); string(b) != "0.0.1\n" {
		t.Fatalf("latest %q after staging 0.0.1", b)
	}
}

// Where site/releases is empty, as in CI, only the installer is staged.
func TestStageReleasesWithoutReleases(t *testing.T) {
	skipWindows(t)
	site, public := fakeSite(t, ""), t.TempDir()
	if out, err := stageReleases(site, public); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := files(t, public); got != "install" {
		t.Fatalf("staged %s, want install", got)
	}
}

// A release the site would serve but site/releases no longer has, or a test
// build named latest, is refused, and nothing is removed.
func TestStageReleasesRefuses(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.2", "0.0.3-test")
	public := t.TempDir()
	os.MkdirAll(filepath.Join(public, "releases", "0.0.1"), 0o755)
	os.WriteFile(filepath.Join(public, "releases", "0.0.1", "SHA256SUMS"), []byte("old"), 0o644)
	out, err := stageReleases(site, public)
	if err == nil || !strings.Contains(out, "releases/0.0.1 is not a release in") {
		t.Fatalf("stale release: err %v, output %q", err, out)
	}
	if _, err := os.Stat(filepath.Join(public, "releases", "0.0.1", "SHA256SUMS")); err != nil {
		t.Fatalf("the stale release was touched: %v", err)
	}

	out, err = stageReleases(site, t.TempDir(), "0.0.3-test")
	if err == nil || !strings.Contains(out, "does not exist or is a test build") {
		t.Fatalf("test build as latest: err %v, output %q", err, out)
	}
}
