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
