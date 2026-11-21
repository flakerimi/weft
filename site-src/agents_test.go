package main

import (
	"fmt"
	"io/fs"
	"net/http/httptest"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The pages and files for agents, as the build made them from skills/ (the
// repository's root), the docs and the catalogue: /skills, /skills/<name>,
// /skills/<name>.md, /llms.txt and /llms-full.txt (pages/skills/,
// agents/llms.ts).

const site = "https://weft.base.al"

// skillNames are the folders of skills/, each a skill with its SKILL.md.
func skillNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("../skills")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("skills/ has no skill")
	}
	sort.Strings(names)
	return names
}

// A page's own scripts beside the shared runtime, and its islands, from routes.json.
func islandsOf(t *testing.T, r builtRoutes, path string) []string {
	t.Helper()
	for _, p := range r.Pages {
		if p.Path == path {
			return p.Islands
		}
	}
	t.Fatalf("routes.json has no page at %s", path)
	return nil
}

// Each skill is a page at /skills/<name> and its file, byte for byte, at
// /skills/<name>.md, served as Markdown. The page has the skill's heading as
// its h1 and its description, links the raw file, and holds to the docs'
// budget, 10 KB brotli, with no script; /skills lists every skill with its
// raw file and the commands that install them.
func TestSkillPages(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()

	index := readPage(t, d, "/skills")
	heading := regexp.MustCompile(`(?m)^# (.+)$`)
	for _, name := range skillNames(t) {
		want, err := os.ReadFile("../skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		res, body := get(t, srv, "/skills/"+name+".md", "Accept-Encoding", "identity")
		if res.StatusCode != 200 || body != string(want) || res.Header.Get("Content-Type") != "text/markdown; charset=utf-8" {
			t.Errorf("GET /skills/%s.md: %d %s, %d bytes; want skills/%s/SKILL.md as it is", name, res.StatusCode, res.Header.Get("Content-Type"), len(body), name)
		}

		html := readPage(t, d, "/skills/"+name)
		h1 := heading.FindSubmatch(want)
		if h1 == nil || !strings.Contains(html, `<h1 class="docs-title">`+string(h1[1])+`</h1>`) || strings.Count(html, "<h1") != 1 {
			t.Errorf("/skills/%s: the h1 is not the skill's heading, once", name)
		}
		description := regexp.MustCompile(`(?m)^description: (.+)$`).FindSubmatch(want)
		if description == nil || !strings.Contains(html, escapeHTML(string(description[1]))) {
			t.Errorf("/skills/%s does not say when to use the skill", name)
		}
		if !strings.Contains(html, `href="/skills/`+name+`.md"`) {
			t.Errorf("/skills/%s does not link its raw file", name)
		}
		if !strings.Contains(html, "curl -fsSL "+site+"/skills/"+name+".md -o ~/.claude/skills/"+name+"/SKILL.md") {
			t.Errorf("/skills/%s does not say how to install it", name)
		}
		if !regexp.MustCompile(`<a [^>]*href="/skills"[^>]*aria-current="true"`).MatchString(html) {
			t.Errorf("/skills/%s: the header's For agents link is not current", name)
		}

		size := brotliSize(t, d, m, "/pages/skills/"+name+"/index.html")
		for _, link := range stylesheetLink.FindAllStringSubmatch(html, -1) {
			size += brotliSize(t, d, m, link[1])
		}
		t.Logf("/skills/%s: %d B brotli", name, size)
		if size > 10*1024 {
			t.Errorf("/skills/%s is %d B brotli, over 10 KB", name, size)
		}
		if islands := islandsOf(t, r, "/skills/"+name); len(islands) != 0 {
			t.Errorf("/skills/%s has the islands %v; a skill's page has none", name, islands)
		}
		for _, tag := range anyScript.FindAllString(html, -1) {
			if !strings.Contains(tag, `type="speculationrules"`) && !strings.Contains(tag, `type="application/json"`) {
				t.Errorf("/skills/%s has %s", name, tag)
			}
		}

		for _, link := range []string{`href="/skills/` + name + `"`, `href="/skills/` + name + `.md"`} {
			if !strings.Contains(index, link) {
				t.Errorf("/skills does not have %s", link)
			}
		}
		if !strings.Contains(index, " "+name+" ") && !strings.Contains(index, " "+name+";") {
			t.Errorf("/skills: the install commands do not fetch %s", name)
		}
	}
	for _, dir := range []string{"~/.claude/skills", ".claude/skills"} {
		if !strings.Contains(index, "do mkdir -p "+dir+"/$s &amp;&amp; curl -fsSL "+site+"/skills/$s.md -o "+dir+"/$s/SKILL.md; done") {
			t.Errorf("/skills has no command that installs the skills into %s", dir)
		}
	}
	size := brotliSize(t, d, m, "/pages/skills/index.html")
	for _, chunk := range islandChunks(t, d, m, islandsOf(t, r, "/skills")) {
		size += brotliSize(t, d, m, chunk)
	}
	t.Logf("/skills: %d B brotli", size)
	if size > 10*1024 {
		t.Errorf("/skills is %d B brotli, over 10 KB", size)
	}
	if !regexp.MustCompile(`<a [^>]*href="/skills"[^>]*aria-current="page"`).MatchString(index) {
		t.Error("/skills: the header's For agents link is not the current page")
	}
}

// escapeHTML is text as Vue's server render writes it.
var escapeHTML = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace

var (
	listLink = regexp.MustCompile(`(?m)^- \[([^\]]+)\]\(([^)]+)\)(?:: (.*))?$`)
	anyLink  = regexp.MustCompile(`\]\((https://weft\.base\.al[^)\s]*)\)|(https://weft\.base\.al[^\s)` + "`" + `]*)`)
)

// resolves says whether a link to the site, https://weft.base.al/…, leads to
// something the build serves: a page, a file of the manifest, or the
// installer and releases served from disk.
func resolves(r builtRoutes, m builtFiles, link string) bool {
	// A link written bare in a sentence may end at its full stop or comma.
	p := strings.TrimRight(strings.TrimPrefix(link, site), ".,;:")
	if i := strings.IndexAny(p, "?#"); i != -1 {
		p = p[:i]
	}
	if p == "" {
		p = "/"
	}
	if p == "/install" || strings.HasPrefix(p, "/releases/") {
		return true
	}
	if _, ok := m.Files[p]; ok {
		return true
	}
	for _, page := range r.Pages {
		if page.Path == p || page.Path == strings.TrimSuffix(p, "/") {
			return true
		}
	}
	return false
}

// /llms.txt follows the llms.txt convention: a title, a summary, then
// sections of links. Docs has every doc with its description, Components
// every catalogue entry with its purpose, page and raw JSON, Skills every
// skill's raw file, and Optional the rest. Every link resolves to the
// build, and it and llms-full.txt are served as plain text.
func TestLLMsTxt(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	srv := httptest.NewServer(siteHandler(t, stagedReleases(t)))
	defer srv.Close()
	_, text := get(t, srv, "/llms.txt", "Accept-Encoding", "identity")
	for _, p := range []string{"/llms.txt", "/llms-full.txt"} {
		res, _ := get(t, srv, p)
		if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("GET %s: %d %s, want 200 text/plain", p, res.StatusCode, res.Header.Get("Content-Type"))
		}
	}

	if !regexp.MustCompile(`^# Weft\n\n> [^\n]+\n`).MatchString(text) {
		t.Errorf("llms.txt does not start with a title and a summary:\n%.200s", text)
	}
	sections := map[string]string{}
	var order []string
	for _, part := range strings.Split(text, "\n## ")[1:] {
		name, body, _ := strings.Cut(part, "\n")
		sections[name] = body
		order = append(order, name)
	}
	if strings.Join(order, ",") != "Docs,Components,Skills,Optional" {
		t.Errorf("llms.txt has the sections %v, want Docs, Components, Skills, Optional", order)
	}
	links := func(section string) map[string]string {
		out := map[string]string{}
		for _, l := range listLink.FindAllStringSubmatch(sections[section], -1) {
			out[l[2]] = l[3]
		}
		return out
	}

	docs := links("Docs")
	for _, doc := range contentDocs(t) {
		b, err := os.ReadFile(docFile(doc.path))
		if err != nil {
			t.Fatal(err)
		}
		description := regexp.MustCompile(`(?m)^description: (.+)$`).FindSubmatch(b)
		if note, ok := docs[site+doc.path]; !ok || description == nil || note != string(description[1]) {
			t.Errorf("llms.txt: Docs does not list %s with its description", doc.path)
		}
	}
	_, entries, _ := readCatalog(t)
	components := links("Components")
	for _, c := range entries {
		page := site + "/components/" + slugOf(c.Name)
		if note, ok := components[page]; !ok || note != c.Purpose+" ([JSON]("+page+".json))" {
			t.Errorf("llms.txt: Components does not list %s with its purpose and its JSON", c.Name)
		}
	}
	skills := links("Skills")
	for _, name := range skillNames(t) {
		if _, ok := skills[site+"/skills/"+name+".md"]; !ok {
			t.Errorf("llms.txt: Skills does not link %s's raw file", name)
		}
	}
	if len(skills) != len(skillNames(t)) {
		t.Errorf("llms.txt lists %d skills for %d in skills/", len(skills), len(skillNames(t)))
	}
	if _, ok := links("Optional")[site+"/llms-full.txt"]; !ok {
		t.Error("llms.txt: Optional does not link llms-full.txt")
	}

	for _, l := range anyLink.FindAllStringSubmatch(text, -1) {
		link := l[1] + l[2]
		if !resolves(r, m, link) {
			t.Errorf("llms.txt links %s, which the build does not serve", link)
		}
	}
}

// docFile is the Markdown file of a doc's route.
func docFile(route string) string {
	slug := strings.TrimPrefix(strings.TrimPrefix(route, "/docs"), "/")
	if slug == "" {
		return "content/docs/index.md"
	}
	if _, err := os.Stat("content/docs/" + slug + ".md"); err == nil {
		return "content/docs/" + slug + ".md"
	}
	return "content/docs/" + slug + "/index.md"
}

// /llms-full.txt has every doc and skill as its file is, after the lines
// naming its page and its file.
func TestLLMsFull(t *testing.T) {
	d := dist(t)
	b, err := fs.ReadFile(d, "llms-full.txt")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	check := func(route, file string) {
		want, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		shown := path.Clean("site-src/" + file)
		marker := fmt.Sprintf("Source: %s%s\nFile: %s\n%s\n\n", site, route, shown, strings.Repeat("=", 80))
		at := strings.Index(text, marker)
		if at == -1 || !strings.HasPrefix(text[at+len(marker):], string(want)) {
			t.Errorf("llms-full.txt does not hold %s as it is, after its source", shown)
		}
	}
	for _, doc := range contentDocs(t) {
		check(doc.path, docFile(doc.path))
	}
	for _, name := range skillNames(t) {
		check("/skills/"+name, "../skills/"+name+"/SKILL.md")
	}
}

// A skill is read away from the site, installed in a skills folder, so it
// links the site in full; every such link leads to something the build
// serves.
func TestSkillLinksResolve(t *testing.T) {
	d := dist(t)
	var m builtFiles
	var r builtRoutes
	readJSON(t, d, "manifest.json", &m)
	readJSON(t, d, "routes.json", &r)
	for _, name := range skillNames(t) {
		b, err := os.ReadFile("../skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range anyLink.FindAllStringSubmatch(string(b), -1) {
			link := l[1] + l[2]
			if strings.Contains(link, "<") {
				continue // a pattern, such as /components/<slug>.json
			}
			if !resolves(r, m, link) {
				t.Errorf("skills/%s/SKILL.md links %s, which the build does not serve", name, link)
			}
		}
	}
}
