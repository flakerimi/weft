package main

import (
	"fmt"
	"go/format"
	"net/url"
	"strconv"
	"strings"
)

// resource is a scaffoldSpec with every name weft scaffold writes.
type resource struct {
	scaffoldSpec
	Module      string // the app's module path
	Type        string // Product
	API         string // products, as weft gen names api.products
	Table, View string // ProductsTable, ProductView
	Auth        bool   // app/modules.go lists auth.Module()
	Session     bool   // the app has web/session.ts
}

func newResource(s scaffoldSpec) *resource {
	return &resource{scaffoldSpec: s, Type: pascal(s.Name), API: camel(s.Plural), Table: pascal(s.Plural) + "Table", View: pascal(s.Name) + "View"}
}

// labelField is the field that names a record (its link in the table, its
// title on its page): the first string field, or id when there is none.
func (r *resource) labelField() string {
	for _, f := range r.Fields {
		if f.Type == "string" {
			return f.Name
		}
	}
	return "id"
}

func (r *resource) usesTime() bool {
	for _, f := range r.Fields {
		if f.Type == "date" || f.Type == "datetime" {
			return true
		}
	}
	return false
}

func (r *resource) model() ([]byte, error) {
	var b strings.Builder
	b.WriteString(wrap("// ", fmt.Sprintf("Package %s keeps the %s: the model, kept in the %s table, and its five routes under /api/%s. weft scaffold wrote it; it is yours to change.", r.Plural, human(r.Plural), r.Plural, r.Plural)))
	fmt.Fprintf(&b, "package %s\n\n", r.Plural)
	if r.usesTime() {
		b.WriteString("import (\n\t\"time\"\n\n\t\"github.com/flakerimi/weft/warp/data\"\n)\n\n")
	} else {
		b.WriteString("import \"github.com/flakerimi/weft/warp/data\"\n\n")
	}
	b.WriteString(wrap("// ", fmt.Sprintf("%s is one %s. Its id and two times come from data.Record and are set by the store.", r.Type, human(r.Name))))
	fmt.Fprintf(&b, "type %s struct {\n\tdata.Record\n", r.Type)
	for _, f := range r.Fields {
		fmt.Fprintf(&b, "\t%s %s `%s`\n", pascal(f.Name), f.goType(), f.tag())
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

func (r *resource) routes() ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n", r.Plural)
	if r.Auth {
		b.WriteString("\t\"github.com/flakerimi/weft/warp/auth\"\n")
	}
	b.WriteString("\t\"github.com/flakerimi/weft/warp/data\"\n)\n\ntype module struct{}\n\n")
	fmt.Fprintf(&b, "func (module) Name() string { return %q }\n\n", r.Plural)
	if r.Auth {
		b.WriteString(wrap("// ", fmt.Sprintf("Register adds the five routes of the %[1]s, each behind its permission: %[2]s:read to list and show, %[2]s:create, %[2]s:edit and %[2]s:delete.", human(r.Plural), r.Plural)))
	} else {
		b.WriteString(wrap("// ", fmt.Sprintf("Register adds the five routes of the %s, open to anyone, since app/modules.go lists no login module.", human(r.Plural))))
	}
	fmt.Fprintf(&b, "func (module) Register(app *warp.App) error {\n\tstore := data.NewStore[%s](data.For(app), %q)\n", r.Type, r.Plural)
	if r.Auth {
		fmt.Fprintf(&b, "\tdata.Resource(app, %q, store, data.PerAction(auth.Per(%q)))\n", "/"+r.Plural, r.Plural)
	} else {
		fmt.Fprintf(&b, "\tdata.Resource(app, %q, store)\n", "/"+r.Plural)
	}
	b.WriteString("\treturn nil\n}\n\n// Module returns the module, for app/modules.go to list.\nfunc Module() warp.Module { return module{} }\n")
	return format.Source([]byte(b.String()))
}

// body is the Go literal of a record's body: every field at sample i.
func (r *resource) body(all [][3]sample, i int) string {
	var parts []string
	for k, f := range r.Fields {
		parts = append(parts, fmt.Sprintf("%q: %s", f.Name, all[k][i].lit))
	}
	return "map[string]any{" + strings.Join(parts, ", ") + "}"
}

// story is the resource's test over HTTP: two records made, listed sorted,
// searched and filtered, one field changed, one record deleted, and, behind
// auth, every route refused without a session. What each list holds is
// worked out here, from the values sent.
func (r *resource) story() ([]byte, error) {
	all := make([][3]sample, len(r.Fields))
	for k, f := range r.Fields {
		all[k], _ = f.samples() // parseField has checked there are some
	}
	// The field the PATCH changes: the first whose third value differs from
	// the first record's and, when unique, from the second's.
	patch := 0
	for k, f := range r.Fields {
		s := all[k]
		if s[2].lit != s[0].lit && (!f.Unique || s[2].lit != s[1].lit) {
			patch = k
			break
		}
	}
	pf := r.Fields[patch]
	usesTime := pf.Type == "date" || pf.Type == "datetime"
	pkg, typ, api := r.Plural, r.Plural+"."+r.Type, "/api/"+r.Plural

	var b strings.Builder
	fmt.Fprintf(&b, "package %s_test\n\nimport (\n\t\"net/http\"\n\t\"path/filepath\"\n\t\"slices\"\n\t\"strconv\"\n\t\"testing\"\n", pkg)
	if usesTime {
		b.WriteString("\t\"time\"\n")
	}
	b.WriteString("\n")
	if r.Auth {
		b.WriteString("\t\"github.com/flakerimi/weft/warp/auth/authtest\"\n")
	}
	fmt.Fprintf(&b, "\t\"github.com/flakerimi/weft/warp/data\"\n\t\"github.com/flakerimi/weft/warp/warptest\"\n\n\t%q\n\t%q\n\n\t_ \"modernc.org/sqlite\"\n)\n\n", r.Module+"/app", r.Module+"/app/"+pkg)
	b.WriteString("// start starts the app's modules over a database file of the test's own.\nfunc start(t *testing.T) *warptest.Client {\n\tt.Helper()\n\treturn warptest.New(t, map[string]string{\n")
	b.WriteString("\t\t\"DB_DRIVER\": \"sqlite\",\n\t\t\"DB_URL\": \"file:\" + filepath.Join(t.TempDir(), \"data\", \"test.db\"),\n")
	if r.Auth {
		b.WriteString("\t\t\"AUTH_SECRET\": \"a secret for tests that is long enough to sign\",\n\t\t\"AUTH_OWNER_EMAIL\": \"owner@example.com\",\n")
	}
	b.WriteString("\t}, app.Modules()...)\n}\n\n")
	fmt.Fprintf(&b, "// idsOf lists the ids of a page of %s, in order.\nfunc idsOf(l data.List[%s]) []int64 {\n\tout := []int64{}\n\tfor _, v := range l.Items {\n\t\tout = append(out, v.ID)\n\t}\n\treturn out\n}\n\n", human(r.Plural), typ)
	fmt.Fprintf(&b, "// idOf is the %s's id as a path segment.\nfunc idOf(v %s) string { return strconv.FormatInt(v.ID, 10) }\n\n", human(r.Name), typ)

	story := fmt.Sprintf("The story over HTTP, as weft scaffold wrote it: two %s are made, listed sorted, searched and filtered; one field of one changes, and the other is deleted.", human(r.Plural))
	if r.Auth {
		story = fmt.Sprintf("The story over HTTP, as weft scaffold wrote it: the owner signs in and makes two %s, lists them sorted, searched and filtered, changes one field of one and deletes the other; with no session every route is refused.", human(r.Plural))
	}
	b.WriteString(wrap("// ", story))
	b.WriteString("func TestStory(t *testing.T) {\n\tc := start(t)\n")
	if r.Auth {
		b.WriteString("\tauthtest.Owner(t, c)\n")
	}
	fmt.Fprintf(&b, "\n\tvar first, second %s\n", typ)
	fmt.Fprintf(&b, "\tc.Post(%q, %s).Want(http.StatusCreated).JSON(&first)\n", api, r.body(all, 0))
	fmt.Fprintf(&b, "\tc.Post(%q, %s).Want(http.StatusCreated).JSON(&second)\n", api, r.body(all, 1))
	b.WriteString("\tif first.ID == 0 || second.ID == 0 || first.ID == second.ID || first.CreatedAt.IsZero() {\n\t\tt.Fatalf(\"POST answered %+v and %+v\", first, second)\n\t}\n\n")

	// The lists: by id both ways, then the first search field's and the
	// first filter field's value of the second record.
	type list struct{ query, want string }
	ids := func(match func(i int) bool) string {
		var w []string
		for i, n := range []string{"first.ID", "second.ID"} {
			if match(i) {
				w = append(w, n)
			}
		}
		return "[]int64{" + strings.Join(w, ", ") + "}"
	}
	lists := []list{{"?sort=id", "[]int64{first.ID, second.ID}"}, {"?sort=-id", "[]int64{second.ID, first.ID}"}}
	for k, f := range r.Fields {
		if f.listTags() == "sort" || f.Type == "enum" || f.Type == "bool" {
			continue
		}
		q := all[k][1].query
		lists = append(lists, list{"?q=" + url.QueryEscape(q) + "&sort=id", ids(func(i int) bool {
			for j, g := range r.Fields {
				if textTypes[g.Type] && strings.Contains(strings.ToLower(all[j][i].query), strings.ToLower(q)) {
					return true
				}
			}
			return false
		})})
		break
	}
	for k, f := range r.Fields {
		if f.Type != "enum" && f.Type != "bool" {
			continue
		}
		v := all[k][1].query
		lists = append(lists, list{"?" + url.QueryEscape("filter["+f.Name+"]") + "=" + url.QueryEscape(v) + "&sort=id", ids(func(i int) bool { return all[k][i].query == v })})
		break
	}
	b.WriteString("\tfor _, tc := range []struct {\n\t\tquery string\n\t\twant  []int64\n\t}{\n")
	for _, l := range lists {
		fmt.Fprintf(&b, "\t\t{%q, %s},\n", l.query, l.want)
	}
	fmt.Fprintf(&b, "\t} {\n\t\tvar l data.List[%s]\n\t\tc.Get(%q + tc.query).Want(http.StatusOK).JSON(&l)\n", typ, api)
	fmt.Fprintf(&b, "\t\tif got := idsOf(l); !slices.Equal(got, tc.want) || l.Total != len(tc.want) {\n\t\t\tt.Errorf(\"GET %s%%s = %%v of %%d, want %%v\", tc.query, got, l.Total, tc.want)\n\t\t}\n\t}\n\n", api)

	field := "changed." + pascal(pf.Name)
	fmt.Fprintf(&b, "\t// Only %s changes.\n\tvar changed %s\n", pf.Name, typ)
	fmt.Fprintf(&b, "\tc.Patch(%q+idOf(first), map[string]any{%q: %s}).Want(http.StatusOK).JSON(&changed)\n", api+"/", pf.Name, all[patch][2].lit)
	fmt.Fprintf(&b, "\tif %s || changed.ID != first.ID || !changed.CreatedAt.Equal(first.CreatedAt) {\n\t\tt.Errorf(\"PATCH %s answered %%+v\", changed)\n\t}\n", all[patch][2].differs(field), pf.Name)
	fmt.Fprintf(&b, "\tvar shown %s\n\tc.Get(%q + idOf(first)).Want(http.StatusOK).JSON(&shown)\n", typ, api+"/")
	fmt.Fprintf(&b, "\tif %s {\n\t\tt.Errorf(\"GET after the PATCH = %%+v\", shown)\n\t}\n\n", all[patch][2].differs("shown."+pascal(pf.Name)))

	fmt.Fprintf(&b, "\tc.Delete(%q + idOf(second)).Want(http.StatusNoContent)\n", api+"/")
	fmt.Fprintf(&b, "\tc.Get(%q + idOf(second)).Want(http.StatusNotFound)\n", api+"/")
	fmt.Fprintf(&b, "\tvar left data.List[%s]\n\tc.Get(%q).Want(http.StatusOK).JSON(&left)\n", typ, api)
	b.WriteString("\tif got := idsOf(left); !slices.Equal(got, []int64{first.ID}) {\n\t\tt.Errorf(\"after the DELETE the list is %v, want only %d\", got, first.ID)\n\t}\n")

	if r.Auth {
		b.WriteString("\n\t// No session: every route is refused.\n\tanon := c.Fork()\n\tfor name, r := range map[string]*warptest.Response{\n")
		fmt.Fprintf(&b, "\t\t\"list\": anon.Get(%q),\n", api)
		fmt.Fprintf(&b, "\t\t\"show\": anon.Get(%q + idOf(first)),\n", api+"/")
		fmt.Fprintf(&b, "\t\t\"create\": anon.Post(%q, %s),\n", api, r.body(all, 1))
		fmt.Fprintf(&b, "\t\t\"update\": anon.Patch(%q+idOf(first), map[string]any{%q: %s}),\n", api+"/", pf.Name, all[patch][0].lit)
		fmt.Fprintf(&b, "\t\t\"remove\": anon.Delete(%q + idOf(first)),\n", api+"/")
		b.WriteString("\t} {\n\t\tif e := r.Want(http.StatusUnauthorized).Error(); e.Code != \"unauthorized\" {\n\t\t\tt.Errorf(\"%s without a session: %+v\", name, e)\n\t\t}\n\t}\n")
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

// wrap is text as comment lines of at most 78 characters, each starting
// with prefix, and a line end after the last.
func wrap(prefix, text string) string {
	var b strings.Builder
	line := prefix
	for _, w := range strings.Fields(text) {
		if line != prefix && len(line)+1+len(w) > 78 {
			b.WriteString(line + "\n")
			line = prefix
		}
		if line != prefix {
			line += " "
		}
		line += w
	}
	b.WriteString(line + "\n")
	return b.String()
}

// ts is s as a TypeScript string in single quotes.
func ts(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

func (r *resource) tableVue() string {
	label := r.labelField()
	var columns, filters, fields []string
	used := map[string]bool{}
	if label == "id" {
		columns = append(columns, "{ key: 'id', label: 'Id', sortable: true, rowHeader: true },")
	}
	for _, f := range r.Fields {
		name := ts(f.Name)
		lbl := ts(capital(human(f.Name)))
		cfg := "{ name: " + name + ", label: " + lbl
		switch f.Type {
		case "text":
			cfg += ", control: 'textarea'"
		case "int":
			cfg += ", step: 1"
		}
		fields = append(fields, cfg+" },")
		if f.Type == "text" {
			continue // long text is read on the record's page
		}
		col := "{ key: " + name + ", label: " + lbl + ", sortable: true"
		switch f.Type {
		case "int", "float":
			col += ", align: 'end', format: number"
			used["number"] = true
		}
		if f.Name == label {
			col += ", rowHeader: true"
		}
		columns = append(columns, col+" },")
		switch f.Type {
		case "enum":
			var opts []string
			for _, c := range f.Choices {
				opts = append(opts, "{ label: "+ts(capital(human(c)))+", value: "+ts(c)+" }")
			}
			filters = append(filters, "{ key: "+name+", label: "+lbl+", items: ["+strings.Join(opts, ", ")+"] },")
		case "bool":
			filters = append(filters, "{ key: "+name+", label: "+lbl+", items: [{ label: 'Yes', value: 'true' }, { label: 'No', value: 'false' }] },")
		}
	}

	var b strings.Builder
	b.WriteString("<script setup vapor lang=\"ts\">\n")
	b.WriteString(wrap("// ", fmt.Sprintf("An island on /%s: the %s from the server, searched, sorted, filtered and paged there, through DataTable, which makes, changes and deletes them in its dialog with the rules the server has. weft scaffold wrote it; it is yours to change.", r.Plural, human(r.Plural))))
	if r.Session {
		b.WriteString("import { onMounted, ref } from 'vue'\n")
	}
	imports := "TableColumn"
	if len(filters) > 0 {
		imports = "DataTableFilter, EntityFieldConfig, TableColumn"
	} else {
		imports = "EntityFieldConfig, TableColumn"
	}
	fmt.Fprintf(&b, "import type { %s } from '@weft/ui'\n", imports)
	if r.Session {
		fmt.Fprintf(&b, "import { api, ApiError, schema, type %s } from '../.weft-types/api.ts'\n", r.Type)
		b.WriteString("import { requireUser, signInAgain, useSession } from '../session.ts'\n\n")
		b.WriteString("// The table asks only once the visitor is known to be signed in.\nconst ready = ref(false)\nconst { session } = useSession()\nonMounted(async () => {\n  ready.value = (await requireUser()) !== null\n})\n\n")
		b.WriteString("// The session can end while the list is open (signed out in another tab):\n// the next page, sort or search is a 401, and the visitor signs in again.\nfunction listFailed(e: unknown) {\n  if (e instanceof ApiError && e.status === 401) signInAgain()\n}\n\n")
	} else {
		fmt.Fprintf(&b, "import { api, schema, type %s } from '../.weft-types/api.ts'\n\n", r.Type)
	}
	if used["number"] {
		b.WriteString("const number = (v: unknown) => (v == null ? '' : (v as number).toLocaleString())\n")
	}
	if len(used) > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "const columns: TableColumn<%s>[] = [\n", r.Type)
	for _, c := range columns {
		b.WriteString("  " + c + "\n")
	}
	b.WriteString("]\n")
	if len(filters) > 0 {
		b.WriteString("const filterDefs: DataTableFilter[] = [\n")
		for _, f := range filters {
			b.WriteString("  " + f + "\n")
		}
		b.WriteString("]\n")
	}
	b.WriteString("const fields: EntityFieldConfig[] = [\n")
	for _, f := range fields {
		b.WriteString("  " + f + "\n")
	}
	b.WriteString("]\n</script>\n\n<template>\n  <DataTable\n")
	if r.Session {
		b.WriteString("    v-if=\"ready\"\n")
	}
	fmt.Fprintf(&b, "    :resource=\"api.%s\"\n    :schema=\"schema.%sInput\"\n    :columns=\"columns\"\n", r.API, r.Type)
	if len(filters) > 0 {
		b.WriteString("    :filter-defs=\"filterDefs\"\n")
	}
	b.WriteString("    :fields=\"fields\"\n")
	fmt.Fprintf(&b, "    name=%q\n", human(r.Name))
	if label != "name" {
		fmt.Fprintf(&b, "    label-key=%q\n", label)
	}
	fmt.Fprintf(&b, "    caption=%q\n    empty-title=%q\n    empty-description=%q\n", capital(human(r.Plural)), "No "+human(r.Plural), "Make one with New "+human(r.Name)+", or change the search.")
	if r.Session {
		b.WriteString("    @error=\"listFailed\"\n")
	}
	b.WriteString("  >\n")
	fmt.Fprintf(&b, "    <template #%s-cell=\"{ row }\">\n      <Link :to=\"`/%s/${row.id}`\">{{ row.%s }}</Link>\n    </template>\n", label, r.Plural, label)
	for _, f := range r.Fields {
		switch f.Type {
		case "enum":
			fmt.Fprintf(&b, "    <template #%s-cell=\"{ row }\">\n      <Badge v-if=\"row.%s\" :label=\"row.%s\" variant=\"soft\" />\n    </template>\n", f.Name, f.Name, f.Name)
		case "bool":
			fmt.Fprintf(&b, "    <template #%s-cell=\"{ row }\">\n      <Badge v-if=\"row.%s != null\" :label=\"row.%s ? 'Yes' : 'No'\" :tone=\"row.%s ? 'success' : 'neutral'\" variant=\"soft\" />\n    </template>\n", f.Name, f.Name, f.Name, f.Name)
		}
	}
	b.WriteString("  </DataTable>\n")
	if r.Session {
		b.WriteString("  <Alert v-else-if=\"session.problem\" tone=\"danger\" :title=\"session.problem\" />\n  <Skeleton v-else height=\"12rem\" />\n")
	}
	b.WriteString("</template>\n")
	return b.String()
}

func (r *resource) viewVue() string {
	label := r.labelField()
	var b strings.Builder
	b.WriteString("<script setup vapor lang=\"ts\">\n")
	b.WriteString(wrap("// ", fmt.Sprintf("An island on the run-time page /%s/[id]: reads the id from the address, asks api.%s.show, and shows the %s, or that there is none with that id. weft scaffold wrote it; it is yours to change.", r.Plural, r.API, human(r.Name))))
	b.WriteString("import { onMounted, ref } from 'vue'\nimport { useParams } from '@weft/ui'\n")
	fmt.Fprintf(&b, "import { api, ApiError, type %s } from '../.weft-types/api.ts'\n", r.Type)
	if r.Session {
		b.WriteString("import { requireUser, useSession } from '../session.ts'\n")
	}
	fmt.Fprintf(&b, "\nconst record = ref<%s | null>(null)\nconst missing = ref(false)\nconst problem = ref('')\nconst loading = ref(true)\n\n", r.Type)
	b.WriteString("// Only an id as the server writes one: digits with no leading zero, so\n// /" + r.Plural + "/0x2a is not record 42.\nfunction idOf(raw: string | undefined): number | null {\n  if (!raw || !/^[1-9][0-9]*$/.test(raw)) return null\n  const id = Number(raw)\n  return Number.isSafeInteger(id) ? id : null\n}\n\n")
	b.WriteString("onMounted(async () => {\n  try {\n")
	if r.Session {
		b.WriteString("    if (!(await requireUser())) {\n      problem.value = useSession().session.problem\n      return\n    }\n")
	}
	fmt.Fprintf(&b, "    const id = idOf(useParams('/%s/:id').id)\n    if (id === null) missing.value = true\n    else record.value = await api.%s.show(id)\n", r.Plural, r.API)
	b.WriteString("  } catch (e) {\n    if (e instanceof ApiError && e.status === 404) missing.value = true\n")
	b.WriteString("    else if (!(e instanceof ApiError)) {\n      console.error(e)\n      problem.value = 'Something went wrong on this page; reload it and try again.'\n    } else problem.value = e.code === 'network' ? 'The server could not be reached; check the connection and try again.' : e.message\n")
	b.WriteString("  } finally {\n    loading.value = false\n  }\n})\n\n")
	b.WriteString("const shown = (v: unknown) => (v == null ? '' : String(v))\nconst number = (v: number | null | undefined) => (v == null ? '' : v.toLocaleString())\nconst moment = (v: string | null | undefined) => (v == null ? '' : new Date(v).toLocaleString())\nconst yesNo = (v: boolean | null | undefined) => (v == null ? '' : v ? 'Yes' : 'No')\n")
	b.WriteString("</script>\n\n<template>\n  <Stack :gap=\"4\">\n    <Skeleton v-if=\"loading\" height=\"8rem\" />\n")
	fmt.Fprintf(&b, "    <EmptyState v-else-if=\"missing\" title=\"No %s has that id.\" />\n", human(r.Name))
	b.WriteString("    <Alert v-else-if=\"problem\" tone=\"danger\" :title=\"problem\" />\n    <Card v-else-if=\"record\">\n")
	if label == "id" {
		fmt.Fprintf(&b, "      <template #header>%s {{ record.id }}</template>\n", capital(human(r.Name)))
	} else {
		fmt.Fprintf(&b, "      <template #header>{{ record.%s }}</template>\n", label)
	}
	b.WriteString("      <Stack :gap=\"2\">\n")
	for _, f := range r.Fields {
		show := "shown"
		switch f.Type {
		case "int", "float":
			show = "number"
		case "bool":
			show = "yesNo"
		}
		fmt.Fprintf(&b, "        <span>%s: {{ %s(record.%s) }}</span>\n", capital(human(f.Name)), show, f.Name)
	}
	b.WriteString("        <span>Made {{ moment(record.created_at) }}, changed {{ moment(record.updated_at) }}</span>\n      </Stack>\n    </Card>\n")
	fmt.Fprintf(&b, "    <Link to=\"/%s\">Back to the %s</Link>\n  </Stack>\n</template>\n", r.Plural, human(r.Plural))
	return b.String()
}

func (r *resource) indexVue() string {
	return fmt.Sprintf(`<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import %[1]s from '../../components/%[1]s.vue'

useHead({ title: %[2]s, description: %[3]s })
</script>

<template>
  <div data-weft-surface="app">
    <Page title=%[4]s>
      <Section>
        <Island when="load" :component="%[1]s" />
      </Section>
    </Page>
  </div>
</template>
`, r.Table, ts(capital(human(r.Plural))), ts("The "+human(r.Plural)+" the app keeps."), strconv.Quote(capital(human(r.Plural))))
}

func (r *resource) recordVue() string {
	return fmt.Sprintf(`<script lang="ts">
%[5]sexport const runtime = true
</script>

<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import %[1]s from '../../components/%[1]s.vue'

useHead({ title: %[2]s, description: %[3]s })
</script>

<template>
  <div data-weft-surface="app">
    <Page title=%[4]s>
      <Section>
        <Island when="load" :component="%[1]s" />
      </Section>
    </Page>
  </div>
</template>
`, r.View, ts(capital(human(r.Name))), ts("One "+human(r.Name)+" the app keeps."), strconv.Quote(capital(human(r.Name))),
		wrap("// ", fmt.Sprintf("The %s are made after the build, so this page is built once, as a shell, and served for every /%s/<id>; its island reads the id.", human(r.Plural), r.Plural)))
}
