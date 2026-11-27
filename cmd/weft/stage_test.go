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
	os.WriteFile(filepath.Join(site, "releases.txt"), []byte("0.0.1\n0.1.0-alpha.1\n"), 0o644)
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
// releases (each with a SHA256SUMS), releases.txt listing those that are not
// test builds, and, unless latest is "", releases/latest.
func fakeSite(t *testing.T, latest string, versions ...string) string {
	t.Helper()
	site := t.TempDir()
	os.WriteFile(filepath.Join(site, "install"), []byte("#!/bin/sh\n"), 0o644)
	list := ""
	for _, v := range versions {
		dir := filepath.Join(site, "releases", v)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(v), 0o644)
		if !strings.HasSuffix(v, "-test") {
			list += v + "\n"
		}
	}
	os.WriteFile(filepath.Join(site, "releases.txt"), []byte(list), 0o644)
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
	if err == nil || !strings.Contains(out, "releases/0.0.1 is not a release to stage") {
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

// Only the published releases (site/releases.txt) and the one being published
// are staged: a folder left in site/releases (a dry run's build, a copy put
// there by hand) is not, nor is it named latest.
func TestStageReleasesStagesOnlyListedReleases(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.9-leftover", "0.0.1", "0.0.2")
	for _, v := range []string{"0.0.9-leftover", "0.0.3"} {
		os.MkdirAll(filepath.Join(site, "releases", v), 0o755)
		os.WriteFile(filepath.Join(site, "releases", v, "SHA256SUMS"), []byte(v), 0o644)
	}
	out := filepath.Join(t.TempDir(), "releases")
	msg, err := stageReleases(site, out)
	if err != nil {
		t.Fatalf("%v: %s", err, msg)
	}
	if got := files(t, out); got != "install,releases/0.0.1/SHA256SUMS,releases/0.0.2/SHA256SUMS" {
		t.Fatalf("staged %s, want the listed releases only", got)
	}
	if !strings.Contains(msg, "names 0.0.9-leftover, which is not staged; leaving latest out") {
		t.Fatalf("no word that latest names a release not staged: %q", msg)
	}
	// The version being published is staged beside them, and named latest.
	next := filepath.Join(t.TempDir(), "releases")
	if msg, err := stageReleases(site, next, "0.0.3"); err != nil {
		t.Fatalf("%v: %s", err, msg)
	}
	if got := files(t, next); got != "install,releases/0.0.1/SHA256SUMS,releases/0.0.2/SHA256SUMS,releases/0.0.3/SHA256SUMS,releases/latest" {
		t.Fatalf("staged %s", got)
	}
	// A staging that holds the leftover is refused, not merged into.
	os.MkdirAll(filepath.Join(next, "releases", "0.0.9-leftover"), 0o755)
	os.WriteFile(filepath.Join(next, "releases", "0.0.9-leftover", "SHA256SUMS"), []byte("0.0.9-leftover"), 0o644)
	if msg, err := stageReleases(site, next, "0.0.3"); err == nil || !strings.Contains(msg, "releases/0.0.9-leftover is not a release to stage") {
		t.Fatalf("a staged leftover: err %v, output %q", err, msg)
	}
}

// The release the site serves now (--live), when site/releases.txt missed it
// (its publish went live, but was not verified in time to be listed), is
// staged beside the listed ones and the new one, so its users keep it; a
// leftover that is neither listed, live nor new still is not.
func TestStageReleasesKeepsTheLiveRelease(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "", "0.0.1", "0.0.2")
	for _, v := range []string{"0.0.3", "0.0.4", "0.0.9-leftover"} {
		os.MkdirAll(filepath.Join(site, "releases", v), 0o755)
		os.WriteFile(filepath.Join(site, "releases", v, "SHA256SUMS"), []byte(v), 0o644)
	}
	if msg, err := stageReleases("--live", "0.0.3", "--check", site, "0.0.4"); err != nil {
		t.Fatalf("check: %v: %s", err, msg)
	}
	out := filepath.Join(t.TempDir(), "releases")
	if msg, err := stageReleases("--live", "0.0.3", site, out, "0.0.4"); err != nil {
		t.Fatalf("%v: %s", err, msg)
	}
	if got := files(t, out); got != "install,releases/0.0.1/SHA256SUMS,releases/0.0.2/SHA256SUMS,releases/0.0.3/SHA256SUMS,releases/0.0.4/SHA256SUMS,releases/latest" {
		t.Fatalf("staged %s, want the listed, the live and the new release", got)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "releases/latest")); string(b) != "0.0.4\n" {
		t.Fatalf("latest %q", b)
	}
	// A live release site/releases does not have is refused, and nothing is staged.
	fresh := filepath.Join(t.TempDir(), "releases")
	msg, err := stageReleases("--live", "0.0.7", site, fresh, "0.0.4")
	if err == nil || strings.TrimSpace(msg) != "stage-releases: "+site+"/releases/0.0.7, the release the site serves now, does not exist or is a test build" {
		t.Fatalf("a live release that is missing: err %v, output %q", err, msg)
	}
	if _, err := os.Stat(fresh); err == nil {
		t.Fatal("staged something after refusing")
	}
}

// Without site/releases.txt nothing says which releases are published, so nothing is staged.
func TestStageReleasesNeedsTheList(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.1", "0.0.1")
	os.Remove(filepath.Join(site, "releases.txt"))
	out := filepath.Join(t.TempDir(), "releases")
	msg, err := stageReleases(site, out)
	if err == nil || strings.TrimSpace(msg) != "stage-releases: "+site+"/releases.txt does not exist; it lists the releases the site has published, the ones staged" {
		t.Fatalf("err %v, output %q", err, msg)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("staged something without the list")
	}
}

// With --link-from, a release already kept is linked from there, after
// checking it is the same; one kept with other contents is refused, and
// nothing is staged.
func TestStageReleasesLinksKeptReleases(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.1", "0.0.2")
	kept := t.TempDir()
	os.MkdirAll(filepath.Join(kept, "0.0.1"), 0o755)
	os.WriteFile(filepath.Join(kept, "0.0.1", "SHA256SUMS"), []byte("0.0.1"), 0o644)
	out := filepath.Join(t.TempDir(), "releases")
	if msg, err := stageReleases("--link-from", kept, site, out); err != nil {
		t.Fatalf("%v: %s", err, msg)
	}
	if got := files(t, out); got != "install,releases/0.0.1/SHA256SUMS,releases/0.0.2/SHA256SUMS,releases/latest" {
		t.Fatalf("staged %s", got)
	}
	if !sameFile(t, filepath.Join(kept, "0.0.1/SHA256SUMS"), filepath.Join(out, "releases/0.0.1/SHA256SUMS")) {
		t.Fatal("the kept release was copied, not linked")
	}
	if sameFile(t, filepath.Join(site, "releases/0.0.2/SHA256SUMS"), filepath.Join(out, "releases/0.0.2/SHA256SUMS")) {
		t.Fatal("a release not kept was linked to the site's")
	}

	os.WriteFile(filepath.Join(kept, "0.0.1", "SHA256SUMS"), []byte("another build"), 0o644)
	fresh := filepath.Join(t.TempDir(), "releases")
	msg, err := stageReleases("--link-from", kept, site, fresh)
	if err == nil || strings.TrimSpace(msg) != "stage-releases: "+site+"/releases/0.0.1 differs from "+kept+"/0.0.1, the release 0.0.1 a verified push served, whose users have its checksums; put "+kept+"/0.0.1 back in "+site+"/releases/0.0.1" {
		t.Fatalf("a kept release that differs: err %v, output %q", err, msg)
	}
	if _, err := os.Stat(fresh); err == nil {
		t.Fatal("staged something after refusing")
	}
}

// A dot-named file in a release (Finder's .DS_Store) is refused, not staged:
// the server would refuse to start with it.
func TestStageReleasesRefusesDotNames(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.1", "0.0.2")
	os.WriteFile(filepath.Join(site, "releases/0.0.1/.DS_Store"), []byte("finder"), 0o644)
	out := filepath.Join(t.TempDir(), "releases")
	msg, err := stageReleases(site, out)
	if err == nil || strings.TrimSpace(msg) != "stage-releases: "+site+"/releases/0.0.1 holds .DS_Store, named with a leading dot, which the site does not serve; move it out" {
		t.Fatalf("err %v, output %q", err, msg)
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("staged something after refusing")
	}
}

func publishedReleases(args ...string) (string, error) {
	b, err := exec.Command("sh", append([]string{"../../scripts/published-releases.sh"}, args...)...).CombinedOutput()
	return string(b), err
}

// Every version in site/releases.txt has been published; a push replaces the
// whole site, so publishing is refused while one of them is not in site/releases.
func TestPublishedReleasesRefusesAMissingOne(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.2")
	os.WriteFile(filepath.Join(site, "releases.txt"), []byte("0.0.1\n0.0.2\n"), 0o644)
	out, err := publishedReleases("check", site)
	if err == nil || !strings.Contains(out, "0.0.1") || strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Fatalf("missing 0.0.1: err %v, output %q (want one sentence naming it)", err, out)
	}

	site = fakeSite(t, "0.0.2", "0.0.1", "0.0.2")
	os.WriteFile(filepath.Join(site, "releases.txt"), []byte("0.0.1\n0.0.2\n"), 0o644)
	if out, err := publishedReleases("check", site); err != nil {
		t.Fatalf("all present: %v: %s", err, out)
	}
}

// After a verified push the new version is added to the list, once.
func TestPublishedReleasesAdd(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "")
	list := filepath.Join(site, "releases.txt")
	os.WriteFile(list, []byte("0.0.1\n0.0.2\n"), 0o644)
	for range 2 {
		if out, err := publishedReleases("add", site, "0.0.3"); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	if b, _ := os.ReadFile(list); string(b) != "0.0.1\n0.0.2\n0.0.3\n" {
		t.Fatalf("releases.txt %q", b)
	}
	if out, err := publishedReleases("add", site, `0.0.4" -X x`); err == nil {
		t.Fatalf("a bad version was added: %s", out)
	}
}

// The version goes into the Dockerfile's -ldflags, so anything that is not a
// version is refused at once, before anything is built or fetched.
func TestPublishSiteRefusesABadVersion(t *testing.T) {
	skipWindows(t)
	for _, v := range []string{`0.0.3" -X evil`, "0.0.3\nRUN x", "v0.0.3", ""} {
		b, err := exec.Command("sh", "../../scripts/publish-site.sh", "--dry-run", v).CombinedOutput()
		out := strings.TrimSpace(string(b))
		if err == nil || !strings.HasPrefix(out, "publish: ") || strings.Contains(out, "\n") {
			t.Errorf("version %q: err %v, output %q (want one sentence)", v, err, out)
		}
	}
}

// release.sh checks the whole argument: a version with a second line is refused
// before anything is built.
func TestReleaseRefusesAMultiLineVersion(t *testing.T) {
	skipWindows(t)
	err := exec.Command("sh", "../../scripts/release.sh", "0.0.3\nRUN x").Run()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 2 {
		t.Fatalf("release.sh with a two-line version: %v, want exit 2", err)
	}
}

// release.sh tests the Go packages it ships before it builds: the root
// package, warp and everything under it, and the CLI.
func TestReleaseTestsWarp(t *testing.T) {
	b, err := os.ReadFile("../../scripts/release.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "\ngo test -short . ./warp/... ./cmd/weft\n") {
		t.Fatal("scripts/release.sh does not run go test -short . ./warp/... ./cmd/weft before it builds")
	}
}

// A staged release that differs from the one in SITE/releases (a file added
// or changed) is refused, not merged, and left as it was.
func TestStageReleasesRefusesADifferentStagedRelease(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "0.0.2", "0.0.1", "0.0.2")
	for _, change := range []func(dir string){
		func(dir string) { os.WriteFile(filepath.Join(dir, "extra.tar.gz"), []byte("old"), 0o644) },
		func(dir string) { os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte("changed"), 0o644) },
	} {
		public := t.TempDir()
		if out, err := stageReleases(site, public); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		change(filepath.Join(public, "releases", "0.0.1"))
		before := files(t, public)
		out, err := stageReleases(site, public)
		msg := strings.TrimSpace(out)
		if err == nil || !strings.Contains(msg, "releases/0.0.1 differs from") || strings.Contains(msg, "\n") {
			t.Fatalf("different staged release: err %v, output %q (want one sentence)", err, out)
		}
		if got := files(t, public); got != before {
			t.Fatalf("staging changed %s\nto       %s", before, got)
		}
	}
}

// A list whose last line has no newline still gets the new version on a line of its own.
func TestPublishedReleasesAddAfterAMissingNewline(t *testing.T) {
	skipWindows(t)
	site := fakeSite(t, "")
	list := filepath.Join(site, "releases.txt")
	os.WriteFile(list, []byte("0.0.1\n0.0.2"), 0o644)
	if out, err := publishedReleases("add", site, "0.0.3"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if b, _ := os.ReadFile(list); string(b) != "0.0.1\n0.0.2\n0.0.3\n" {
		t.Fatalf("releases.txt %q", b)
	}
}

// fakeRepo is a copy of the repository's publishing scripts in a folder of
// their own, with stand-ins for the slow steps (release.sh, site.sh) and for
// the network (curl, lisa and gh on PATH), so publish and rollback run end to
// end without building, fetching or pushing anything real. It is a git
// checkout with one commit, whose origin is a bare repository of the test's
// own, so tags are made and pushed for real, but only there; gh is told of a
// repository that does not exist. weft.base.al starts out serving 0.0.2;
// site/releases.txt lists 0.0.1 and 0.0.2.
type fakeRepo struct {
	root, fake, tmp string
	env             []string
}

func newFakeRepo(t *testing.T) *fakeRepo {
	t.Helper()
	r := &fakeRepo{root: t.TempDir(), fake: t.TempDir(), tmp: t.TempDir()}
	write := func(dir, name, data string, mode os.FileMode) {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(data), mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []string{"publish-site.sh", "rollback-site.sh", "publish-site-static.sh", "version.sh", "published-releases.sh", "wait-live.sh", "stage-releases.sh", "link-release.sh", "github-release.sh"} {
		b, err := os.ReadFile(filepath.Join("../../scripts", s))
		if err != nil {
			t.Fatal(err)
		}
		write(r.root, "scripts/"+s, string(b), 0o755)
	}
	// release.sh and site.sh make what the real ones make, from nothing.
	write(r.root, "scripts/release.sh", `#!/bin/sh
set -eu
out="${2:-site/releases}"
mkdir -p "$out/$1"
[ -z "${RELEASE_FAILS:-}" ] || { echo "release: go build failed" >&2; exit 1; }
echo "tarball of $1" >"$out/$1/weft_linux_amd64.tar.gz"
echo "sums-of-$1  weft_linux_amd64.tar.gz" >"$out/$1/SHA256SUMS"
echo "$1" >"$out/latest"
echo "$1" >>"$FAKE/built"
`, 0o755)
	write(r.root, "scripts/site.sh", `#!/bin/sh
set -eu
echo "${WEFT_SITE_VERSION:-}" >"$FAKE/site-version"
mkdir -p site-src/dist
echo "<p>home</p>" >site-src/dist/index.html
`, 0o755)
	// The Go of the repository as it is laid out: the root package holds
	// Version, warp (with a package under it, as later phases add) is what
	// site-src/main.go imports, and each has a test the tree must leave out.
	write(r.root, "go.mod", "module github.com/flakerimi/weft\n\ngo 1.26\n", 0o644)
	write(r.root, "version.go", "package weft\n\nvar Version = \"0.0.0-dev\"\n", 0o644)
	write(r.root, "version_test.go", "package weft\n\nimport \"testing\"\n\nfunc TestNothing(t *testing.T) { t.Fatal(\"a test file was pushed\") }\n", 0o644)
	write(r.root, "warp/warp.go", "package warp\n\nimport \"github.com/flakerimi/weft/warp/data\"\n\nfunc Name() string { return data.Name }\n", 0o644)
	write(r.root, "warp/warp_test.go", "package warp\n\nthis does not compile\n", 0o644)
	write(r.root, "warp/data/data.go", "package data\n\nconst Name = \"warp\"\n", 0o644)
	write(r.root, "warp/data/data_test.go", "package data\n\nthis does not compile\n", 0o644)
	write(r.root, "site-src/main.go", `package main

import (
	"fmt"

	"github.com/flakerimi/weft"
	"github.com/flakerimi/weft/warp"
)

func main() { fmt.Println(weft.Version, warp.Name()) }
`, 0o644)
	write(r.root, "site/releases.txt", "0.0.1\n0.0.2\n", 0o644)
	write(r.root, "site/install", "#!/bin/sh\n", 0o644)
	for _, v := range []string{"0.0.1", "0.0.2"} {
		write(r.root, "site/releases/"+v+"/SHA256SUMS", "sums of "+v+"\n", 0o644)
		write(r.root, "site/releases/"+v+"/weft_linux_amd64.tar.gz", "tarball of "+v+"\n", 0o644)
	}
	// curl answers from the fake's state; lisa logs its arguments and serves
	// the version the pushed Dockerfile builds.
	bin := filepath.Join(r.fake, "bin")
	write(bin, "curl", `#!/bin/sh
for a; do url=$a; done
case "$url" in
*/releases/latest) cat "$FAKE/latest" ;;
*) cat "$FAKE/home" ;;
esac
`, 0o755)
	write(bin, "lisa", `#!/bin/sh
echo "$*" >>"$FAKE/lisa.log"
[ -z "${PUSH_FAILS:-}" ] || { echo "lisa: the server refused the push" >&2; exit 3; }
[ "$1 $2 $3 $4" = "server push --name weft" ] && [ -f "$5/Dockerfile" ] || exit 2
sed -n 's/.*weft.Version=\([^ "]*\).*/\1/p' "$5/Dockerfile" >"$FAKE/latest"
(cd "$5" && find . -type f | sort) >"$FAKE/pushed-files"
# What origin has at the push: the tag must be there first.
v=$(cat "$FAKE/latest")
if git --git-dir="$FAKE/origin.git" rev-parse -q --verify "refs/tags/v$v" >/dev/null; then
	echo "push $v, tagged on origin" >>"$FAKE/order"
else
	echo "push $v, not tagged on origin" >>"$FAKE/order"
fi
echo '<weft-island data-id="CopyCommand" data-when="load">' >"$FAKE/home"
# Something takes the place the tree would be kept in.
[ -z "${BLOCK_KEEP:-}" ] || echo blocked >"$BLOCK_KEEP"
`, 0o755)
	// gh logs its arguments; it is signed in unless GH_SIGNED_OUT is set, and
	// knows the releases it made, and those $FAKE/release-<tag> marks.
	write(bin, "gh", `#!/bin/sh
echo "$*" >>"$FAKE/gh.log"
case "$1 $2" in
"auth status") [ -z "${GH_SIGNED_OUT:-}" ] ;;
"release view") [ -e "$FAKE/release-$3" ] ;;
"release create")
	[ -z "${GH_CREATE_FAILS:-}" ] || { echo "gh: HTTP 502" >&2; exit 1; }
	echo "release $3" >>"$FAKE/order"
	touch "$FAKE/release-$3"
	;;
*) exit 2 ;;
esac
`, 0o755)
	write(r.fake, "latest", "0.0.2\n", 0o644)
	write(r.fake, "home", `<weft-island data-id="CopyCommand" data-when="load">`, 0o644)
	write(r.root, ".gitignore", "site/releases/\nsite/pushed/\nsite-src/dist/\n", 0o644)
	gitConfig := filepath.Join(r.fake, "gitconfig")
	write(r.fake, "gitconfig", "", 0o644)
	r.env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FAKE="+r.fake, "TMPDIR="+r.tmp,
		"WEFT_GITHUB_REPO=example.invalid/no-such-repo",
		"GIT_CONFIG_GLOBAL="+gitConfig, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
	r.git(t, "init", "-q", "--bare", filepath.Join(r.fake, "origin.git"))
	r.git(t, "init", "-q")
	r.git(t, "remote", "add", "origin", filepath.Join(r.fake, "origin.git"))
	r.git(t, "add", "-A")
	r.git(t, "commit", "-q", "-m", "a repository")
	return r
}

// git runs git in the fake repository, with its environment.
func (r *fakeRepo) git(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = r.root, r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *fakeRepo) fakeFile(name string) string {
	b, _ := os.ReadFile(filepath.Join(r.fake, name))
	return string(b)
}

// run runs one of the repository's scripts; it returns the output, trimmed.
func (r *fakeRepo) run(script string, args ...string) (string, error) {
	cmd := exec.Command("sh", append([]string{filepath.Join(r.root, "scripts", script)}, args...)...)
	cmd.Env = r.env
	b, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(b)), err
}

func (r *fakeRepo) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(r.root, name))
	return string(b)
}

func (r *fakeRepo) lisa() string {
	b, _ := os.ReadFile(filepath.Join(r.fake, "lisa.log"))
	return string(b)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

// A verified push keeps the tree it pushed as site/pushed/<version>, one per
// version, and leaves no tree behind in TMPDIR or beside the kept ones (only
// each publish's release build, in TMPDIR, for the owner to trash). The
// tree's binary embeds the pages; its releases go into the image beside it.
func TestPublishSiteKeepsOneTreePerVersion(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	pushed := filepath.Join(r.root, "site", "pushed")
	for _, v := range []string{"0.0.3", "0.0.4"} {
		if out, err := r.run("publish-site.sh", v); err != nil {
			t.Fatalf("publish %s: %v\n%s", v, err, out)
		}
		if got := lastLine(r.lisa()); !strings.HasPrefix(got, "server push --name weft "+pushed+"/.weft-site-push.") {
			t.Fatalf("publish %s pushed %q, want a tree made in site/pushed", v, got)
		}
		// The pages are built for the version published, which /index.md names.
		if b, _ := os.ReadFile(filepath.Join(r.fake, "site-version")); strings.TrimSpace(string(b)) != v {
			t.Errorf("publish %s built the site with WEFT_SITE_VERSION %q", v, b)
		}
		pushedFiles, _ := os.ReadFile(filepath.Join(r.fake, "pushed-files"))
		var kept []string
		for _, f := range strings.Split(files(t, filepath.Join(pushed, v)), ",") {
			kept = append(kept, "./"+f)
		}
		if got := strings.Join(kept, "\n") + "\n"; got != string(pushedFiles) {
			t.Fatalf("publish %s kept\n%s\nbut pushed\n%s", v, got, pushedFiles)
		}
		df := r.read("site/pushed/" + v + "/Dockerfile")
		if !strings.Contains(df, "weft.Version="+v+`"`) || !strings.Contains(df, "COPY site-src/releases /data/releases\nRUN chmod -R u=rwX,go=rX /data/releases\n") ||
			!strings.Contains(df, "COPY --from=build /data/releases /data/releases\n") ||
			strings.Contains(df, "COPY . .") || !strings.Contains(df, "COPY warp warp\n") ||
			strings.Index(df, "COPY warp warp\n") > strings.Index(df, "go build") {
			t.Fatalf("site/pushed/%s/Dockerfile:\n%s", v, df)
		}
		if got := r.read("site/pushed/" + v + "/site-src/releases/releases/latest"); got != v+"\n" {
			t.Fatalf("site/pushed/%s serves latest %q", v, got)
		}
	}
	got := files(t, pushed)
	for _, want := range []string{"0.0.3/site-src/releases/install", "0.0.3/site-src/releases/releases/0.0.1/SHA256SUMS",
		"0.0.3/site-src/releases/releases/0.0.3/SHA256SUMS", "0.0.4/site-src/releases/releases/0.0.4/SHA256SUMS", "0.0.4/go.mod",
		"0.0.4/site-src/dist/index.html", "0.0.4/site-src/main.go", "0.0.4/version.go", "0.0.4/warp/warp.go", "0.0.4/warp/data/data.go"} {
		if !strings.Contains(","+got+",", ","+want+",") {
			t.Errorf("site/pushed lacks %s", want)
		}
	}
	if strings.Contains(got, "_test.go") || strings.Contains(got, "site-src/dist/releases") || strings.Contains(got, "site-src/dist/install") {
		t.Fatalf("site/pushed holds %s", got)
	}
	var names []string
	entries, _ := os.ReadDir(pushed)
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "0.0.3,0.0.4,releases" {
		t.Fatalf("site/pushed holds %v, want 0.0.3, 0.0.4 and releases", names)
	}
	// TMPDIR keeps each publish's release build, which the publish never
	// deletes (the owner trashes it, from the path it prints), and nothing else.
	left, _ := os.ReadDir(r.tmp)
	for _, e := range left {
		if !strings.HasPrefix(e.Name(), "weft-release.") {
			t.Errorf("TMPDIR keeps %s", e.Name())
		}
	}
	if len(left) != 2 {
		t.Fatalf("TMPDIR keeps %v, want the two release builds", left)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n0.0.4\n" {
		t.Fatalf("releases.txt %q", got)
	}
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	fa, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	return os.SameFile(fa, fb)
}

// Each release a verified push served is kept once, in site/pushed/releases:
// every kept tree holds hard links to it, never a copy of its own, and never
// a link to site/releases, where a rebuild would change it in every tree.
func TestPublishSiteKeepsEachReleaseOnce(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	for _, v := range []string{"0.0.3", "0.0.4"} {
		if out, err := r.run("publish-site.sh", v); err != nil {
			t.Fatalf("publish %s: %v\n%s", v, err, out)
		}
	}
	pushed := filepath.Join(r.root, "site", "pushed")
	var want []string
	for _, v := range []string{"0.0.1", "0.0.2", "0.0.3", "0.0.4"} {
		want = append(want, v+"/SHA256SUMS", v+"/weft_linux_amd64.tar.gz")
	}
	if got := files(t, filepath.Join(pushed, "releases")); got != strings.Join(want, ",") {
		t.Fatalf("site/pushed/releases holds %s", got)
	}
	// Kept read-only, so a write through any of its links fails.
	for _, f := range want {
		if fi, _ := os.Stat(filepath.Join(pushed, "releases", f)); fi.Mode().Perm()&0o222 != 0 {
			t.Errorf("site/pushed/releases/%s is %v, writable", f, fi.Mode().Perm())
		}
	}
	for _, v := range []string{"0.0.1", "0.0.2", "0.0.3", "0.0.4"} {
		kept := filepath.Join(pushed, "releases", v, "SHA256SUMS")
		for _, tree := range []string{"0.0.3", "0.0.4"} {
			if tree == "0.0.3" && v == "0.0.4" {
				continue
			}
			if !sameFile(t, kept, filepath.Join(pushed, tree, "site-src/releases/releases", v, "SHA256SUMS")) {
				t.Errorf("the tree kept for %s holds its own copy of release %s", tree, v)
			}
		}
		if sameFile(t, kept, filepath.Join(r.root, "site/releases", v, "SHA256SUMS")) {
			t.Errorf("release %s is kept as a link to site/releases", v)
		}
	}

	// A kept release rebuilt in site/releases, in place, changes no tree, and
	// the next publish is refused: its users have the kept checksums.
	sums := filepath.Join(r.root, "site/releases/0.0.3/SHA256SUMS")
	f, err := os.OpenFile(sums, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("rebuilt\n")
	f.Close()
	if got := r.read("site/pushed/0.0.3/site-src/releases/releases/0.0.3/SHA256SUMS"); got != "sums-of-0.0.3  weft_linux_amd64.tar.gz\n" {
		t.Fatalf("the kept tree's release changed to %q", got)
	}
	pushes := r.lisa()
	out, err := r.run("publish-site.sh", "0.0.5")
	if err == nil || !strings.Contains(lastLine(out), "stage-releases: site/releases/0.0.3 differs from site/pushed/releases/0.0.3") {
		t.Fatalf("publish after a kept release changed: err %v\n%s", err, out)
	}
	if r.lisa() != pushes {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
	// Refused before anything is built or a tree is made.
	if b, _ := os.ReadFile(filepath.Join(r.fake, "built")); strings.Contains(string(b), "0.0.5") {
		t.Fatalf("release.sh built 0.0.5 before the refusal")
	}
	if left, _ := filepath.Glob(filepath.Join(pushed, ".weft-site-push.*")); len(left) != 0 {
		t.Fatalf("a refused publish left %v", left)
	}
}

// A tarball changed in place through a kept tree's link (the checksums file
// left as it was) is not pushed again: rollback compares whole releases.
func TestRollbackSiteRefusesAChangedTarball(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("publish: %v\n%s", err, out)
	}
	pushes := r.lisa()
	tarball := filepath.Join(r.root, "site/pushed/0.0.3/site-src/releases/releases/0.0.1/weft_linux_amd64.tar.gz")
	os.Chmod(tarball, 0o644)
	f, err := os.OpenFile(tarball, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("corrupt\n")
	f.Close()
	out, err := r.run("rollback-site.sh", "0.0.3")
	if err == nil || out != "rollback: the tree kept for 0.0.3 does not serve the published release 0.0.1 as site/releases has it, so pushing it would drop or change that release; publish a new version with scripts/publish-site.sh instead." {
		t.Fatalf("err %v, output %q", err, out)
	}
	if r.lisa() != pushes {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// A version is listed as published the moment its push succeeds: a tree
// that cannot then be kept leaves it listed, and says so.
func TestPublishSiteListsAPushedVersionWhoseTreeIsNotKept(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	r.env = append(r.env, "BLOCK_KEEP="+filepath.Join(r.root, "site/pushed/0.0.3"))
	out, err := r.run("publish-site.sh", "0.0.3")
	msg := lastLine(out)
	if err == nil || !strings.HasPrefix(msg, "publish: 0.0.3 is live and listed in site/releases.txt, but its tree ") {
		t.Fatalf("err %v, output\n%s", err, out)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n" {
		t.Fatalf("releases.txt %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.root, "site/pushed/releases")); err == nil {
		t.Fatal("releases were kept for a tree that was not")
	}
}

// failWaitLive makes wait-live.sh time out, as when the server is slow to
// build the image: the push itself succeeded. Each change is committed, as a
// publish builds only from a commit.
func (r *fakeRepo) failWaitLive(t *testing.T) func() {
	t.Helper()
	p := filepath.Join(r.root, "scripts/wait-live.sh")
	real, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, []byte("#!/bin/sh\necho \"wait-live: after 5 minutes it is not live\" >&2\nexit 1\n"), 0o755)
	r.git(t, "commit", "-q", "-am", "wait-live times out")
	return func() {
		os.WriteFile(p, real, 0o755)
		r.git(t, "commit", "-q", "-am", "wait-live as it was")
	}
}

// A pushed release counts as published, so it is listed when its push
// succeeds, not after wait-live: when wait-live then times out it stays
// listed, and the next publish stages it. Chained: X (0.0.3) is live but not
// listed; Y (0.0.4) stages and lists it, then times out itself; Z (0.0.5)
// still stages X, Y and Z.
func TestPublishSiteKeepsReleasesThroughChainedTimeouts(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.MkdirAll(filepath.Join(r.root, "site/releases/0.0.3"), 0o755)
	os.WriteFile(filepath.Join(r.root, "site/releases/0.0.3/SHA256SUMS"), []byte("sums of 0.0.3\n"), 0o644)
	os.WriteFile(filepath.Join(r.fake, "latest"), []byte("0.0.3\n"), 0o644)

	restore := r.failWaitLive(t)
	out, err := r.run("publish-site.sh", "0.0.4")
	restore()
	if err == nil || !strings.Contains(lastLine(out), "publish: 0.0.4 was pushed and is listed in site/releases.txt (commit it), so the next publish stages it, but it is not verified live (see above)") {
		t.Fatalf("a timed-out publish: err %v, output\n%s", err, out)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n0.0.4\n" {
		t.Fatalf("after the timeout, releases.txt %q", got)
	}

	if out, err := r.run("publish-site.sh", "0.0.5"); err != nil {
		t.Fatalf("publish 0.0.5: %v\n%s", err, out)
	}
	pushed, _ := os.ReadFile(filepath.Join(r.fake, "pushed-files"))
	for _, v := range []string{"0.0.1", "0.0.2", "0.0.3", "0.0.4", "0.0.5"} {
		if !strings.Contains(string(pushed), "./site-src/releases/releases/"+v+"/SHA256SUMS") {
			t.Errorf("publish 0.0.5 dropped release %s:\n%s", v, pushed)
		}
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n0.0.4\n0.0.5\n" {
		t.Fatalf("releases.txt %q", got)
	}
}

// A real publish that cannot read the live release refuses before building
// anything: it cannot tell which release it must keep.
func TestPublishSiteRefusesAnUnreachableSite(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.Remove(filepath.Join(r.fake, "latest"))
	out, err := r.run("publish-site.sh", "0.0.3")
	if err == nil || out != "publish: cannot read the live release from https://weft.base.al/releases/latest, so cannot check that it is kept" {
		t.Fatalf("err %v, output %q", err, out)
	}
	if b, err := os.ReadFile(filepath.Join(r.fake, "built")); err == nil {
		t.Fatalf("release.sh ran for %q", b)
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// When the site already serves the version being published (its earlier push
// was listed by no one, and site/releases does not have it), it is published
// as new, with no word of an unlisted release, and listed once.
func TestPublishSiteWhenLiveIsTheNewVersion(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.WriteFile(filepath.Join(r.fake, "latest"), []byte("0.0.3\n"), 0o644)
	out, err := r.run("publish-site.sh", "0.0.3")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "without being listed") {
		t.Fatalf("the new version was reported as an unlisted live release:\n%s", out)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n" {
		t.Fatalf("releases.txt %q", got)
	}
}

// When the site serves a listed release, nothing is said about it and nothing extra is staged.
func TestPublishSiteWhenLiveIsListed(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	out, err := r.run("publish-site.sh", "0.0.3")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "without being listed") {
		t.Fatalf("a listed live release was reported as unlisted:\n%s", out)
	}
	pushed, _ := os.ReadFile(filepath.Join(r.fake, "pushed-files"))
	if got := strings.Count(string(pushed), "/SHA256SUMS"); got != 3 {
		t.Fatalf("pushed %d releases, want 0.0.1 to 0.0.3:\n%s", got, pushed)
	}
}

// A version site/releases already has was released before: building it again
// would change the tarballs its users have checksums for. Refused, in one
// sentence, before anything is built, even when the site serves it.
func TestPublishSiteRefusesAReleasedVersion(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.MkdirAll(filepath.Join(r.root, "site/releases/0.0.3"), 0o755)
	os.WriteFile(filepath.Join(r.root, "site/releases/0.0.3/SHA256SUMS"), []byte("sums of 0.0.3\n"), 0o644)
	want := "publish: 0.0.3 is already released (site/releases/0.0.3 exists), and building it again would change its tarballs; publish a new version."
	for _, live := range []string{"0.0.2", "0.0.3"} {
		os.WriteFile(filepath.Join(r.fake, "latest"), []byte(live+"\n"), 0o644)
		for _, args := range [][]string{{"0.0.3"}, {"--dry-run", "0.0.3"}} {
			out, err := r.run("publish-site.sh", args...)
			if err == nil || out != want {
				t.Fatalf("live %s, %v: err %v, output %q", live, args, err, out)
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(r.fake, "built")); err == nil {
		t.Fatalf("release.sh ran for %q", b)
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// before is the state of site/releases and site/releases.txt a failed publish must leave as it found it.
func (r *fakeRepo) before(t *testing.T) string {
	t.Helper()
	return files(t, filepath.Join(r.root, "site/releases")) + "|" + r.read("site/releases.txt")
}

// A real publish builds its release in TMPDIR, as a dry run does, and copies
// it into site/releases only once the push has succeeded: a push that fails
// leaves site/releases and its list as they were, says where the build is,
// and the same version can be published again at once.
func TestPublishSiteFailedPushLeavesSiteReleasesAlone(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	was := r.before(t)
	env := r.env
	r.env = append(env, "PUSH_FAILS=1")
	out, err := r.run("publish-site.sh", "0.0.3")
	r.env = env
	commit := r.git(t, "rev-parse", "HEAD")
	if err == nil || !strings.Contains(out, "publish: stopped before 0.0.3 was pushed; site/releases and site/releases.txt are as they were, and the tag v0.0.3 is on origin, naming "+commit+" (publishing 0.0.3 again from that commit uses it). This run's release build is in "+r.tmp+"/weft-release.") {
		t.Fatalf("a failed push: err %v, output\n%s", err, out)
	}
	if now := r.before(t); now != was {
		t.Fatalf("a failed push changed site/releases:\nwas %s\nnow %s", was, now)
	}
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("retrying 0.0.3 after a failed push: %v\n%s", err, out)
	}
}

// A release build that fails leaves no folder in site/releases.
func TestPublishSiteFailedReleaseBuildLeavesNoFolder(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	was := r.before(t)
	r.env = append(r.env, "RELEASE_FAILS=1")
	out, err := r.run("publish-site.sh", "0.0.3")
	if err == nil || !strings.Contains(out, "publish: stopped before 0.0.3 was pushed") {
		t.Fatalf("a failed release build: err %v, output\n%s", err, out)
	}
	if now := r.before(t); now != was {
		t.Fatalf("a failed release build changed site/releases:\nwas %s\nnow %s", was, now)
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// Once pushed, the release is copied into site/releases, byte for byte what
// the push staged, and named latest; the build folder is left for the owner,
// and its path printed.
func TestPublishSiteCopiesThePushedRelease(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	out, err := r.run("publish-site.sh", "0.0.3")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	staged := filepath.Join(r.root, "site/pushed/0.0.3/site-src/releases/releases/0.0.3")
	copied := filepath.Join(r.root, "site/releases/0.0.3")
	if got, want := files(t, copied), files(t, staged); got != want || got == "" {
		t.Fatalf("site/releases/0.0.3 holds %s, the push staged %s", got, want)
	}
	for _, f := range strings.Split(files(t, staged), ",") {
		a, _ := os.ReadFile(filepath.Join(staged, f))
		b, _ := os.ReadFile(filepath.Join(copied, f))
		if string(a) != string(b) {
			t.Errorf("site/releases/0.0.3/%s differs from what was pushed", f)
		}
	}
	if got := r.read("site/releases/latest"); got != "0.0.3\n" {
		t.Fatalf("site/releases/latest %q", got)
	}
	builds, _ := filepath.Glob(filepath.Join(r.tmp, "weft-release.*"))
	if len(builds) != 1 || !strings.Contains(out, builds[0]) {
		t.Fatalf("the build folder %v is not named in the output:\n%s", builds, out)
	}
}

// When the list cannot be written after a push, the publish says which
// version to add to site/releases.txt by hand.
func TestPublishSiteSaysWhenTheListCannotBeWritten(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	list := filepath.Join(r.root, "site/releases.txt")
	os.Chmod(list, 0o444)
	defer os.Chmod(list, 0o644)
	out, err := r.run("publish-site.sh", "0.0.3")
	if err == nil || !strings.Contains(out, "publish: 0.0.3 was pushed, but site/releases.txt could not be written; add the line 0.0.3 to it by hand, so the next publish keeps 0.0.3.") {
		t.Fatalf("err %v, output\n%s", err, out)
	}
}

// A version already published is refused before anything is built: building
// it again would change the tarballs its users have checksums for.
func TestPublishSiteRefusesAPublishedVersion(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	for _, args := range [][]string{{"--dry-run", "0.0.2"}, {"0.0.1"}} {
		out, err := r.run("publish-site.sh", args...)
		if err == nil || !strings.HasPrefix(out, "publish: ") || !strings.Contains(out, "site/releases.txt") || strings.Contains(out, "\n") {
			t.Fatalf("%v: err %v, output %q (want one sentence)", args, err, out)
		}
	}
	if b, err := os.ReadFile(filepath.Join(r.fake, "built")); err == nil {
		t.Fatalf("release.sh ran for %q", b)
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// A dry run stops before the push and keeps nothing under site/pushed.
func TestPublishSiteDryRunPushesNothing(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	out, err := r.run("publish-site.sh", "--dry-run", "0.0.3")
	if err != nil || !strings.Contains(out, "dry run: not pushed") {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "dry run: built 0.0.3 in "+r.tmp+"/weft-release.") || !strings.Contains(out, "site/releases is as it was") {
		t.Fatalf("the dry run does not say where it built 0.0.3:\n%s", out)
	}
	// It leaves site/releases as it was, so 0.0.3 can be published for real after it.
	if _, err := os.Stat(filepath.Join(r.root, "site/releases/0.0.3")); err == nil {
		t.Fatal("the dry run left site/releases/0.0.3")
	}
	if _, err := os.Stat(filepath.Join(r.root, "site/releases/latest")); err == nil {
		t.Fatal("the dry run wrote site/releases/latest")
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
	if _, err := os.Stat(filepath.Join(r.root, "site", "pushed")); err == nil {
		t.Fatal("a dry run kept a tree under site/pushed")
	}
	trees, _ := filepath.Glob(filepath.Join(r.tmp, "weft-site-push.*"))
	if len(trees) != 1 || !strings.Contains(out, "the tree is "+trees[0]) {
		t.Fatalf("dry run trees %v, output\n%s", trees, out)
	}
	if got := files(t, filepath.Join(trees[0], "site-src/releases")); got != "install,releases/0.0.1/SHA256SUMS,releases/0.0.1/weft_linux_amd64.tar.gz,releases/0.0.2/SHA256SUMS,releases/0.0.2/weft_linux_amd64.tar.gz,releases/0.0.3/SHA256SUMS,releases/0.0.3/weft_linux_amd64.tar.gz,releases/latest" {
		t.Fatalf("the dry run's tree stages %s", got)
	}
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("publishing 0.0.3 after its dry run: %v\n%s", err, out)
	}
}

// The push tree holds all the Go the site's binary needs, warp and the
// packages under it included, and no test: it compiles with nothing but the
// tree, as the server compiles it.
func TestPublishSiteTreeCompiles(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	if out, err := r.run("publish-site.sh", "--dry-run", "0.0.3"); err != nil {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	trees, _ := filepath.Glob(filepath.Join(r.tmp, "weft-site-push.*"))
	if len(trees) != 1 {
		t.Fatalf("dry run trees %v", trees)
	}
	got := files(t, trees[0])
	if strings.Contains(got, "_test.go") || !strings.Contains(","+got+",", ",warp/data/data.go,") {
		t.Fatalf("the tree holds %s", got)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "weft-site"), "./site-src")
	build.Dir = trees[0]
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off", "GOWORK=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./site-src in the push tree: %v\n%s", err, out)
	}
}

// A publish keeps the release the site serves now, even when site/releases.txt
// missed it (pushed by a publish from before a push was listed at once), says
// so, and lists it when the new one is pushed; a leftover folder that is
// neither listed, live nor new is not pushed.
func TestPublishSiteKeepsALiveUnlistedRelease(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	for _, v := range []string{"0.0.3", "0.0.9-leftover"} {
		os.MkdirAll(filepath.Join(r.root, "site/releases", v), 0o755)
		os.WriteFile(filepath.Join(r.root, "site/releases", v, "SHA256SUMS"), []byte("sums of "+v+"\n"), 0o644)
	}
	os.WriteFile(filepath.Join(r.fake, "latest"), []byte("0.0.3\n"), 0o644)
	out, err := r.run("publish-site.sh", "0.0.4")
	if err != nil {
		t.Fatalf("publish: %v\n%s", err, out)
	}
	if !strings.Contains(out, "publish: https://weft.base.al serves 0.0.3, which was pushed earlier without being listed in site/releases.txt; it is staged so its users keep it, and listed when 0.0.4 is pushed") {
		t.Fatalf("the publish does not say 0.0.3 was missing from site/releases.txt:\n%s", out)
	}
	pushed, _ := os.ReadFile(filepath.Join(r.fake, "pushed-files"))
	if !strings.Contains(string(pushed), "./site-src/releases/releases/0.0.3/SHA256SUMS") {
		t.Fatalf("the live release 0.0.3 was not pushed:\n%s", pushed)
	}
	if strings.Contains(string(pushed), "0.0.9-leftover") {
		t.Fatalf("the leftover was pushed:\n%s", pushed)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n0.0.4\n" {
		t.Fatalf("releases.txt %q", got)
	}
}

// The static site cannot come back: Lisa keeps the weft app a container.
func TestPublishSiteStaticRefuses(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	out, err := r.run("publish-site-static.sh", "0.0.3")
	if err == nil || !strings.Contains(out, "scripts/rollback-site.sh") || strings.Contains(out, "\n") {
		t.Fatalf("err %v, output %q (want one sentence naming the rollback)", err, out)
	}
	if r.lisa() != "" {
		t.Fatalf("lisa ran: %s", r.lisa())
	}
}

// A rollback re-pushes a kept tree, but only one that still serves every
// published release.
func TestRollbackSite(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	for _, v := range []string{"0.0.3", "0.0.4"} {
		if out, err := r.run("publish-site.sh", v); err != nil {
			t.Fatalf("publish %s: %v\n%s", v, err, out)
		}
	}
	pushes := strings.Count(r.lisa(), "\n")
	refused := func(why string, args ...string) {
		t.Helper()
		out, err := r.run("rollback-site.sh", args...)
		if err == nil || !strings.Contains(out, why) || strings.Contains(out, "\n") {
			t.Fatalf("%v: err %v, output %q (want one sentence with %q)", args, err, out, why)
		}
		if n := strings.Count(r.lisa(), "\n"); n != pushes {
			t.Fatalf("%v: lisa ran: %s", args, r.lisa())
		}
	}
	refused("rollback: give a version", "0.0.3\" -X x")
	refused("rollback: site/pushed/0.0.9 holds no pushed tree", "0.0.9")
	// 0.0.3's tree has no 0.0.4, which is published: pushing it would drop that.
	refused("rollback: the tree kept for 0.0.3 does not serve the published release 0.0.4", "0.0.3")
	// published-releases.sh check runs first.
	os.Rename(filepath.Join(r.root, "site/releases/0.0.1"), filepath.Join(r.tmp, "0.0.1"))
	refused("published-releases: site/releases lacks the published release(s) 0.0.1", "0.0.4")
	os.Rename(filepath.Join(r.tmp, "0.0.1"), filepath.Join(r.root, "site/releases/0.0.1"))

	// A dry run names the tree and pushes nothing.
	if out, err := r.run("rollback-site.sh", "--dry-run", "0.0.4"); err != nil || !strings.Contains(out, "site/pushed/0.0.4") {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	if n := strings.Count(r.lisa(), "\n"); n != pushes {
		t.Fatalf("a dry run ran lisa: %s", r.lisa())
	}

	// Once 0.0.4 is no longer listed (its push was never verified), 0.0.3 may come back.
	os.WriteFile(filepath.Join(r.root, "site/releases.txt"), []byte("0.0.1\n0.0.2\n0.0.3\n"), 0o644)
	out, err := r.run("rollback-site.sh", "0.0.3")
	if err != nil {
		t.Fatalf("rollback: %v\n%s", err, out)
	}
	if got := lastLine(r.lisa()); got != "server push --name weft "+filepath.Join(r.root, "site/pushed/0.0.3") {
		t.Fatalf("rollback pushed %q", got)
	}
	if b, _ := os.ReadFile(filepath.Join(r.fake, "latest")); strings.TrimSpace(string(b)) != "0.0.3" {
		t.Fatalf("the site serves %q", b)
	}
}

// A tree kept before releases moved out of the binary (as 0.0.3's was) has
// them in its embedded build, site-src/dist/releases; it rolls back the same.
func TestRollbackSiteToATreeThatEmbedsItsReleases(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	tree := filepath.Join(r.root, "site/pushed/0.0.2")
	for name, data := range map[string]string{
		"Dockerfile": "RUN go build -ldflags \"-X github.com/flakerimi/weft.Version=0.0.2\" ./site-src\n",
		"site-src/dist/releases/0.0.1/SHA256SUMS":              "sums of 0.0.1\n",
		"site-src/dist/releases/0.0.1/weft_linux_amd64.tar.gz": "tarball of 0.0.1\n",
		"site-src/dist/releases/0.0.2/SHA256SUMS":              "sums of 0.0.2\n",
		"site-src/dist/releases/0.0.2/weft_linux_amd64.tar.gz": "tarball of 0.0.2\n",
	} {
		os.MkdirAll(filepath.Dir(filepath.Join(tree, name)), 0o755)
		os.WriteFile(filepath.Join(tree, name), []byte(data), 0o644)
	}
	if out, err := r.run("rollback-site.sh", "0.0.2"); err != nil {
		t.Fatalf("rollback: %v\n%s", err, out)
	}
	if got := lastLine(r.lisa()); got != "server push --name weft "+tree {
		t.Fatalf("rollback pushed %q", got)
	}
	// And one that lacks a published release is refused, as a new tree is.
	os.WriteFile(filepath.Join(tree, "site-src/dist/releases/0.0.1/SHA256SUMS"), []byte("other\n"), 0o644)
	out, err := r.run("rollback-site.sh", "0.0.2")
	if err == nil || out != "rollback: the tree kept for 0.0.2 does not serve the published release 0.0.1 as site/releases has it, so pushing it would drop or change that release; publish a new version with scripts/publish-site.sh instead." {
		t.Fatalf("err %v, output %q", err, out)
	}
}

// A publish tags the commit it builds from and pushes the tag before the
// site, then makes the GitHub Release from the files the site serves: the
// archives SHA256SUMS lists, and SHA256SUMS.
func TestPublishSiteTagsAndReleases(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	commit := r.git(t, "rev-parse", "HEAD")
	out, err := r.run("publish-site.sh", "0.0.3")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := r.git(t, "cat-file", "-t", "v0.0.3"); got != "tag" {
		t.Errorf("v0.0.3 is a %s, want an annotated tag", got)
	}
	if got := r.git(t, "rev-parse", "v0.0.3^{commit}"); got != commit {
		t.Errorf("v0.0.3 names %s, want HEAD %s", got, commit)
	}
	if got := r.git(t, "--git-dir", filepath.Join(r.fake, "origin.git"), "rev-parse", "v0.0.3^{commit}"); got != commit {
		t.Errorf("origin's v0.0.3 names %s, want %s", got, commit)
	}
	if got := r.fakeFile("order"); got != "push 0.0.3, tagged on origin\nrelease v0.0.3\n" {
		t.Errorf("the order was %q, want the tag on origin before the push, and the release after it", got)
	}
	dir := "site/releases/0.0.3" // github-release.sh runs at the repository's root
	want := "release create v0.0.3 --repo example.invalid/no-such-repo --verify-tag --title v0.0.3 --notes " + releaseNotes("0.0.3") + " " +
		dir + "/weft_linux_amd64.tar.gz " + dir + "/SHA256SUMS"
	if log := r.fakeFile("gh.log"); !strings.Contains(log, "\n"+want+"\n") {
		t.Errorf("gh was run as\n%s\nwant a line\n%s", log, want)
	}
	if log := r.fakeFile("gh.log"); !strings.HasPrefix(log, "auth status --hostname github.com\n") || strings.Contains(log, "auth status\n") {
		t.Errorf("gh auth status is not asked of github.com alone:\n%s", log)
	}
	if got := lastLine(out); got != "published the GitHub Release v0.0.3" {
		t.Errorf("last line %q", got)
	}
	// A tag of another version on the same commit is no matter.
	if out, err := r.run("publish-site.sh", "0.0.4"); err != nil {
		t.Fatalf("0.0.4 from the same commit: %v\n%s", err, out)
	}
}

// A tag of the version on another commit, here or on origin, stops the
// publish before anything is built or pushed.
func TestPublishSiteRefusesATagElsewhere(t *testing.T) {
	skipWindows(t)
	for _, where := range []string{"here", "origin"} {
		t.Run(where, func(t *testing.T) {
			r := newFakeRepo(t)
			old := r.git(t, "rev-parse", "HEAD")
			r.git(t, "tag", "-a", "v0.0.3", "-m", "elsewhere")
			if where == "origin" {
				r.git(t, "push", "-q", "origin", "refs/tags/v0.0.3")
				r.git(t, "tag", "-d", "v0.0.3")
			}
			os.WriteFile(filepath.Join(r.root, "go.mod"), []byte("module github.com/flakerimi/weft\n\ngo 1.26\n\n// changed\n"), 0o644)
			r.git(t, "commit", "-q", "-am", "a later commit")
			now := r.git(t, "rev-parse", "HEAD")
			was := r.before(t)
			out, err := r.run("publish-site.sh", "0.0.3")
			want := "publish: the tag v0.0.3 already names " + old + ", not " + now
			if where == "origin" {
				want = "publish: origin already has the tag v0.0.3 on " + old + ", not on " + now
			}
			if err == nil || !strings.HasPrefix(out, want) || strings.Contains(out, "\n") {
				t.Fatalf("err %v, output %q, want one sentence starting %q", err, out, want)
			}
			if b := r.fakeFile("built"); b != "" {
				t.Fatalf("release.sh ran for %q", b)
			}
			if r.lisa() != "" || r.before(t) != was || strings.Contains(r.fakeFile("gh.log"), "release create") {
				t.Fatal("the refused publish pushed or changed something")
			}
		})
	}
}

// A GitHub Release of the version stops the publish before anything is
// built, tagged or pushed.
func TestPublishSiteRefusesAnExistingRelease(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.WriteFile(filepath.Join(r.fake, "release-v0.0.3"), nil, 0o644)
	out, err := r.run("publish-site.sh", "0.0.3")
	if err == nil || out != "publish: the GitHub Release v0.0.3 already exists, and a publish never replaces one; publish a new version." {
		t.Fatalf("err %v, output %q", err, out)
	}
	if r.fakeFile("built") != "" || r.lisa() != "" || r.git(t, "tag", "-l", "v0.0.3") != "" {
		t.Fatal("the refused publish built, pushed or tagged")
	}
}

// Tracked files that differ from HEAD stop a publish, since the tag would
// not name what is built; a dry run says so and goes on.
func TestPublishSiteRefusesChangedFiles(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.WriteFile(filepath.Join(r.root, "warp/data/data.go"), []byte("package data\n\nconst Name = \"changed\"\n"), 0o644)
	commit := r.git(t, "rev-parse", "HEAD")
	out, err := r.run("publish-site.sh", "0.0.3")
	want := "publish: tracked files differ from " + commit + ", which the tag v0.0.3 would name, so the module at the tag would not be what this release is built from; commit them, or set them aside, and publish again."
	if err == nil || out != want {
		t.Fatalf("err %v, output %q, want %q", err, out, want)
	}
	if r.fakeFile("built") != "" || r.lisa() != "" {
		t.Fatal("the refused publish built or pushed")
	}
	out, err = r.run("publish-site.sh", "--dry-run", "0.0.3")
	if err != nil || !strings.Contains(out, "publish: tracked files differ from "+commit+", which a publish refuses") {
		t.Fatalf("dry run: err %v, output\n%s", err, out)
	}
}

// With gh signed out, the site is still published and tagged; the publish
// says, after the push and again at the end, what to run, and fails.
func TestPublishSiteWithoutGh(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	r.env = append(r.env, "GH_SIGNED_OUT=1")
	out, err := r.run("publish-site.sh", "0.0.3")
	say := "publish: 0.0.3 is pushed and tagged v0.0.3, but gh is not signed in to github.com, so its GitHub Release is not made; run gh auth login, then scripts/github-release.sh 0.0.3."
	end := "publish: 0.0.3 is live and tagged v0.0.3, but its GitHub Release is not made, since gh is not signed in to github.com; run gh auth login, then scripts/github-release.sh 0.0.3."
	if err == nil || !strings.Contains(out, say) || lastLine(out) != end {
		t.Fatalf("err %v, output\n%s", err, out)
	}
	if got := r.fakeFile("order"); got != "push 0.0.3, tagged on origin\n" {
		t.Errorf("order %q", got)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n0.0.3\n" || !exists(t, filepath.Join(r.root, "site/pushed/0.0.3")) {
		t.Errorf("the publish did not finish its own steps: releases.txt %q", got)
	}
	// The command it names makes the release once gh is signed in.
	r.env = r.env[:len(r.env)-1]
	if out, err := r.run("github-release.sh", "0.0.3"); err != nil || !strings.HasPrefix(out, "made the GitHub Release v0.0.3") {
		t.Fatalf("github-release.sh: %v\n%s", err, out)
	}
	if got := r.fakeFile("order"); got != "push 0.0.3, tagged on origin\nrelease v0.0.3\n" {
		t.Errorf("order %q", got)
	}
}

// A push that fails leaves the tag on origin; publishing the version again
// from the same commit uses it, and pushes no other.
func TestPublishSiteUsesItsOwnTag(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	env := r.env
	r.env = append(env, "PUSH_FAILS=1")
	if _, err := r.run("publish-site.sh", "0.0.3"); err == nil {
		t.Fatal("the failed push succeeded")
	}
	r.env = env
	out, err := r.run("publish-site.sh", "--dry-run", "0.0.3")
	commit := r.git(t, "rev-parse", "HEAD")
	if err != nil || !strings.Contains(out, "dry run: the tag v0.0.3 names "+commit+" already; a publish would not make it again") ||
		!strings.Contains(out, "dry run: origin has the tag v0.0.3 on "+commit+" already; a publish would not push it again") {
		t.Fatalf("dry run: err %v, output\n%s", err, out)
	}
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("again: %v\n%s", err, out)
	}
	if got := r.fakeFile("order"); got != "push 0.0.3, tagged on origin\nrelease v0.0.3\n" {
		t.Errorf("order %q", got)
	}
}

// A dry run tags, pushes and releases nothing, and prints the git and gh
// commands a publish would run.
func TestPublishSiteDryRunPrintsTheCommands(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	commit := r.git(t, "rev-parse", "HEAD")
	out, err := r.run("publish-site.sh", "--dry-run", "0.0.3")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{
		"dry run: would run, before the site push: git tag -a v0.0.3 -m 'Weft 0.0.3' " + commit + "\n",
		"dry run: would run, before the site push: git push origin refs/tags/v0.0.3\n",
		"dry run: would run: gh release create v0.0.3 --repo example.invalid/no-such-repo --verify-tag --title v0.0.3 --notes '" + releaseNotes("0.0.3") + "' site/releases/0.0.3/weft_linux_amd64.tar.gz site/releases/0.0.3/SHA256SUMS\n",
	} {
		if !strings.Contains(out+"\n", want) {
			t.Errorf("the dry run does not print\n%s\nin\n%s", want, out)
		}
	}
	if r.git(t, "tag", "-l") != "" || r.git(t, "--git-dir", filepath.Join(r.fake, "origin.git"), "tag", "-l") != "" {
		t.Error("the dry run made or pushed a tag")
	}
	if strings.Contains(r.fakeFile("gh.log"), "release create") || r.fakeFile("order") != "" {
		t.Error("the dry run made a release or pushed")
	}
}

// github-release.sh attaches exactly what SHA256SUMS lists, marks a
// pre-release, and refuses a list that names a file not there.
func TestGithubRelease(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	dir := "site/releases/0.1.0-alpha.1" // as github-release.sh names it, from the repository's root
	abs := filepath.Join(r.root, dir)
	os.MkdirAll(abs, 0o755)
	for _, f := range []string{"weft_darwin_arm64.tar.gz", "weft_linux_amd64.tar.gz", "weft_windows_amd64.zip", "stray.txt"} {
		os.WriteFile(filepath.Join(abs, f), []byte(f), 0o644)
	}
	os.WriteFile(filepath.Join(abs, "SHA256SUMS"), []byte("aa  weft_darwin_arm64.tar.gz\nbb *weft_linux_amd64.tar.gz\ncc  weft_windows_amd64.zip\n"), 0o644)
	out, err := r.run("github-release.sh", "0.1.0-alpha.1")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	log := r.fakeFile("gh.log")
	files := " " + dir + "/weft_darwin_arm64.tar.gz " + dir + "/weft_linux_amd64.tar.gz " + dir + "/weft_windows_amd64.zip " + dir + "/SHA256SUMS\n"
	if !strings.Contains(log, " --prerelease"+files) || strings.Contains(log, "stray.txt") {
		t.Errorf("gh was run as\n%s", log)
	}
	// Made once: a second run finds it and stops.
	out, err = r.run("github-release.sh", "0.1.0-alpha.1")
	if err == nil || out != "github-release: the GitHub Release v0.1.0-alpha.1 already exists in example.invalid/no-such-repo, and this script never replaces one; leave it, or remove it by hand and run scripts/github-release.sh 0.1.0-alpha.1 again." {
		t.Errorf("again: err %v, output %q", err, out)
	}
	os.WriteFile(filepath.Join(abs, "SHA256SUMS"), []byte("aa  weft_gone.tar.gz\n"), 0o644)
	out, err = r.run("github-release.sh", "0.1.0-alpha.1")
	if err == nil || out != "github-release: "+dir+"/SHA256SUMS lists weft_gone.tar.gz, which is not in "+dir+"; copy the release weft.base.al serves for 0.1.0-alpha.1 there." {
		t.Errorf("a missing file: err %v, output %q", err, out)
	}
}

// releaseNotes is the GitHub Release's notes for version: every link in it
// is one weft.base.al serves once the version is pushed.
func releaseNotes(version string) string {
	return "weft " + version + " for darwin and linux, on amd64 and arm64: the files weft.base.al serves, with their checksums at https://weft.base.al/releases/" + version + "/SHA256SUMS. Install it as https://weft.base.al/docs/getting-started/install says. An app this release makes requires github.com/flakerimi/weft v" + version + "."
}

// Files git neither tracks nor ignores, in what a release or the site is
// built from, stop a publish: they would be built and pushed, but not be in
// the tag. Elsewhere, such as the owner's notes at the root, they are no
// matter; a dry run says so and goes on.
func TestPublishSiteRefusesUntrackedFiles(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.WriteFile(filepath.Join(r.root, "coord.md"), []byte("notes\n"), 0o644)
	for _, f := range []string{"warp/new.go", "warp/data/more.go", "templates/app/extra.tmpl", "packages/ui/x.ts"} {
		os.MkdirAll(filepath.Dir(filepath.Join(r.root, f)), 0o755)
		os.WriteFile(filepath.Join(r.root, f), []byte("x\n"), 0o644)
	}
	out, err := r.run("publish-site.sh", "0.0.3")
	want := "publish: files git does not track are in what the release and the site are built from (packages/ui/x.ts, templates/app/extra.tmpl, warp/data/more.go, …), so the tag v0.0.3 would not name what is built; commit them, or move them out, and publish again."
	if err == nil || out != want {
		t.Fatalf("err %v, output %q, want %q", err, out, want)
	}
	if r.fakeFile("built") != "" || r.lisa() != "" || r.git(t, "tag", "-l") != "" {
		t.Fatal("the refused publish built, pushed or tagged")
	}
	out, err = r.run("publish-site.sh", "--dry-run", "0.0.3")
	if err != nil || !strings.Contains(out, "publish: files git does not track are in what the release and the site are built from (packages/ui/x.ts, templates/app/extra.tmpl, warp/data/more.go, …), which a publish refuses") {
		t.Fatalf("dry run: err %v, output\n%s", err, out)
	}
	// Moved out, only the owner's notes are left untracked, and that is no matter.
	for _, d := range []string{"warp/new.go", "warp/data/more.go", "templates", "packages"} {
		os.Rename(filepath.Join(r.root, d), filepath.Join(t.TempDir(), "moved"))
	}
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("with only coord.md untracked: %v\n%s", err, out)
	}
}

// The home page embeds the newest benchmark results file, so an untracked one
// would go live without being in the tag: it stops a publish too.
func TestPublishSiteRefusesUntrackedBenchResults(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	os.MkdirAll(filepath.Join(r.root, "bench/results"), 0o755)
	os.WriteFile(filepath.Join(r.root, "bench/results/2099-01-01.json"), []byte("{}\n"), 0o644)
	out, err := r.run("publish-site.sh", "0.0.3")
	want := "publish: files git does not track are in what the release and the site are built from (bench/results/2099-01-01.json), so the tag v0.0.3 would not name what is built; commit them, or move them out, and publish again."
	if err == nil || out != want {
		t.Fatalf("err %v, output %q, want %q", err, out, want)
	}
	if r.git(t, "tag", "-l") != "" {
		t.Fatal("the refused publish tagged")
	}
}

// A tag origin has and the checkout does not (a fresh clone) is fetched,
// not made again: the checkout's tag is origin's object, so a later fetch
// of the tags does not refuse to clobber it.
func TestPublishSiteFetchesOriginsTag(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	r.git(t, "tag", "-a", "v0.0.3", "-m", "made elsewhere")
	r.git(t, "push", "-q", "origin", "refs/tags/v0.0.3")
	theirs := r.git(t, "rev-parse", "v0.0.3")
	r.git(t, "tag", "-d", "v0.0.3") // the test's own repository: this checkout never had it
	out, err := r.run("publish-site.sh", "--dry-run", "0.0.3")
	if err != nil || !strings.Contains(out, "dry run: would run, before the site push: git fetch origin refs/tags/v0.0.3:refs/tags/v0.0.3\n") {
		t.Fatalf("dry run: err %v, output\n%s", err, out)
	}
	if out, err := r.run("publish-site.sh", "0.0.3"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if mine := r.git(t, "rev-parse", "v0.0.3"); mine != theirs {
		t.Errorf("the checkout's v0.0.3 is %s, origin's %s", mine, theirs)
	}
	r.git(t, "fetch", "-q", "--tags", "origin")
}

// gh not installed: the publish says to install it, not to sign in.
func TestPublishSiteWithoutGhInstalled(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	// The fakes but gh, and the system's own tools; no gh of the machine's.
	bin := t.TempDir()
	for _, f := range []string{"curl", "lisa"} {
		os.Symlink(filepath.Join(r.fake, "bin", f), filepath.Join(bin, f))
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	os.Symlink(git, filepath.Join(bin, "git"))
	for i, kv := range r.env {
		if strings.HasPrefix(kv, "PATH=") {
			r.env[i] = "PATH=" + bin + ":/usr/bin:/bin:/usr/sbin:/sbin"
		}
	}
	out, err := r.run("publish-site.sh", "0.0.3")
	say := "publish: 0.0.3 is pushed and tagged v0.0.3, but gh is not installed, so its GitHub Release is not made; install gh (https://cli.github.com), run gh auth login, then scripts/github-release.sh 0.0.3."
	end := "publish: 0.0.3 is live and tagged v0.0.3, but its GitHub Release is not made, since gh is not installed; install gh (https://cli.github.com), run gh auth login, then scripts/github-release.sh 0.0.3."
	if err == nil || !strings.Contains(out, say) || lastLine(out) != end {
		t.Fatalf("err %v, output\n%s", err, out)
	}
}
