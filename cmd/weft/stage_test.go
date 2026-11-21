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
// the network (curl and lisa on PATH), so publish and rollback run end to end
// without building, fetching or pushing anything real. weft.base.al starts out
// serving 0.0.2; site/releases.txt lists 0.0.1 and 0.0.2.
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
	for _, s := range []string{"publish-site.sh", "rollback-site.sh", "publish-site-static.sh", "version.sh", "published-releases.sh", "wait-live.sh", "stage-releases.sh", "link-release.sh"} {
		b, err := os.ReadFile(filepath.Join("../../scripts", s))
		if err != nil {
			t.Fatal(err)
		}
		write(r.root, "scripts/"+s, string(b), 0o755)
	}
	// release.sh and site.sh make what the real ones make, from nothing.
	write(r.root, "scripts/release.sh", `#!/bin/sh
set -eu
mkdir -p "site/releases/$1"
echo "sums of $1" >"site/releases/$1/SHA256SUMS"
echo "tarball of $1" >"site/releases/$1/weft_linux_amd64.tar.gz"
echo "$1" >site/releases/latest
echo "$1" >>"$FAKE/built"
`, 0o755)
	write(r.root, "scripts/site.sh", `#!/bin/sh
set -eu
mkdir -p site-src/dist
echo "<p>home</p>" >site-src/dist/index.html
`, 0o755)
	write(r.root, "go.mod", "module github.com/flakerimi/weft\n", 0o644)
	write(r.root, "weft.go", "package weft\n", 0o644)
	write(r.root, "weft_test.go", "package weft\n", 0o644)
	write(r.root, "site-src/main.go", "package main\n", 0o644)
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
[ "$1 $2 $3 $4" = "server push --name weft" ] && [ -f "$5/Dockerfile" ] || exit 2
sed -n 's/.*weft.Version=\([^ "]*\).*/\1/p' "$5/Dockerfile" >"$FAKE/latest"
(cd "$5" && find . -type f | sort) >"$FAKE/pushed-files"
echo '<weft-island data-id="CopyCommand" data-when="load">' >"$FAKE/home"
# Something takes the place the tree would be kept in.
[ -z "${BLOCK_KEEP:-}" ] || echo blocked >"$BLOCK_KEEP"
`, 0o755)
	write(r.fake, "latest", "0.0.2\n", 0o644)
	write(r.fake, "home", `<weft-island data-id="CopyCommand" data-when="load">`, 0o644)
	r.env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FAKE="+r.fake, "TMPDIR="+r.tmp)
	return r
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
// version, and leaves no tree behind in TMPDIR or beside the kept ones. The
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
			strings.Contains(df, "COPY . .") {
			t.Fatalf("site/pushed/%s/Dockerfile:\n%s", v, df)
		}
		if got := r.read("site/pushed/" + v + "/site-src/releases/releases/latest"); got != v+"\n" {
			t.Fatalf("site/pushed/%s serves latest %q", v, got)
		}
	}
	got := files(t, pushed)
	for _, want := range []string{"0.0.3/site-src/releases/install", "0.0.3/site-src/releases/releases/0.0.1/SHA256SUMS",
		"0.0.3/site-src/releases/releases/0.0.3/SHA256SUMS", "0.0.4/site-src/releases/releases/0.0.4/SHA256SUMS", "0.0.4/go.mod",
		"0.0.4/site-src/dist/index.html", "0.0.4/site-src/main.go"} {
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
	if left, _ := os.ReadDir(r.tmp); len(left) != 0 {
		t.Fatalf("TMPDIR keeps %v", left)
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
	if got := r.read("site/pushed/0.0.3/site-src/releases/releases/0.0.3/SHA256SUMS"); got != "sums of 0.0.3\n" {
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

// The tree is kept before the version is listed: a version listed as
// published always has a tree to roll back to.
func TestPublishSiteListsOnlyAKeptTree(t *testing.T) {
	skipWindows(t)
	r := newFakeRepo(t)
	r.env = append(r.env, "BLOCK_KEEP="+filepath.Join(r.root, "site/pushed/0.0.3"))
	out, err := r.run("publish-site.sh", "0.0.3")
	msg := lastLine(out)
	if err == nil || !strings.HasPrefix(msg, "publish: 0.0.3 is live, but its tree ") || !strings.Contains(msg, "so 0.0.3 is not listed as published") {
		t.Fatalf("err %v, output\n%s", err, out)
	}
	if got := r.read("site/releases.txt"); got != "0.0.1\n0.0.2\n" {
		t.Fatalf("releases.txt %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.root, "site/pushed/releases")); err == nil {
		t.Fatal("releases were kept for a tree that was not")
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
	if !strings.Contains(out, "dry run: built 0.0.3 into site/releases/0.0.3 and named it in site/releases/latest, locally") {
		t.Fatalf("the dry run does not say it left site/releases/0.0.3 behind:\n%s", out)
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
}

// A publish keeps the release the site serves now, even when site/releases.txt
// missed it (its publish went live after wait-live gave up), says so, and
// lists it once the new one is verified live; a leftover folder that is
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
	if !strings.Contains(out, "publish: https://weft.base.al serves 0.0.3, which site/releases.txt does not list; it is staged so its users keep it, and listed once 0.0.4 is verified live") {
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
