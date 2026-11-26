package warp

// The scan of a typed route's body: one pass over bytes encoding/json has
// already decoded into the plan's view, which finds the top-level keys the
// body sent, refuses a readonly key at any depth, and finds every null sent
// for a value that cannot be null (anything but a pointer, an interface or a
// type that reads its own JSON), which encoding/json would leave as zero.
//
// It mirrors encoding/json v1's matching of a key to a field (decode.go,
// object): the field whose key is the key itself, else the one whose key
// folds the same (foldName), the key unquoted as unquoteBytes does.
// Registration refuses two keys of one object that fold the same
// (foldOwners), so a key sent in any case names at most one field, and the
// choice between folded fields that encoding/json v1 and jsonv2 make
// differently (the first, the shallowest) never arises; the scan reads each
// value once, as that one field. TestScanMatchesEncodingJSON and FuzzScan
// hold it to encoding/json. Should the two part all the same, what holds is
// plan.readonly: every readonly field is put back to zero after decoding, so
// a readonly field is never set, whatever the scan said.

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// object is what may be sent where a struct is: the json keys of its
// fields in field order, which of them are readonly, and how to look inside
// each one's value, nil where nothing inside it can be readonly. exact finds
// the field of a key as sent; folded the one whose key is the same once
// folded, of which registration allows one.
type object struct {
	names    []string
	readonly []bool
	inner    []*look
	exact    map[string]int
	folded   map[string]int
	longest  int // the length of the longest folded name, in bytes
}

// index makes o's exact and folded maps from its names.
func (o *object) index() {
	o.exact = make(map[string]int, len(o.names))
	o.folded = make(map[string]int, len(o.names))
	for i, name := range o.names {
		o.exact[name] = i
		f := string(appendFolded(nil, []byte(name)))
		if _, taken := o.folded[f]; !taken { // registration refuses a second
			o.folded[f] = i
		}
		o.longest = max(o.longest, len(f))
	}
}

// look is how to look inside a JSON value: as a struct's object, as a list
// of values or as an object of values (a map), each value looked at so.
// notNull is whether the value may not be null; size, for a fixed-size
// array, how many elements its list must have, -1 for any other value.
type look struct {
	obj        *object
	list, dict *look
	notNull    bool
	size       int
}

// objects makes the objects of the struct types a body reaches, each once,
// so that a type that holds itself ends.
type objects map[reflect.Type]*object

func (m objects) lookFor(t reflect.Type) *look {
	nullable := false
	for t.Kind() == reflect.Pointer && !decodesItself(t) {
		t, nullable = t.Elem(), true
	}
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		return nil // nil is a value of its own
	}
	l := &look{notNull: !nullable, size: -1}
	switch {
	case decodesItself(t):
		// Not looked inside. One that reads its own JSON is described as
		// unknown, and is handed null; time.Time and one that reads itself
		// as text are described as text, and encoding/json leaves them zero.
		if t != timeType && (t.Implements(jsonUnmarshaler) || reflect.PointerTo(t).Implements(jsonUnmarshaler)) {
			l.notNull = false
		}
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Array:
		l.list = m.lookFor(t.Elem())
		if t.Kind() == reflect.Array && !isBytes(t) {
			l.size = t.Len()
		}
	case t.Kind() == reflect.Map:
		l.dict = m.lookFor(t.Elem())
	case t.Kind() == reflect.Struct && t != timeType:
		l.obj = m.objectOf(t)
	}
	if !l.notNull && l.size < 0 && l.obj == nil && l.list == nil && l.dict == nil {
		return nil
	}
	return l
}

func (m objects) objectOf(t reflect.Type) *object {
	if o, ok := m[t]; ok {
		return o
	}
	o := &object{}
	m[t] = o
	fields, _ := inputFields(t, nil)
	for _, f := range fields {
		key, _, ok := jsonTagOf(f.StructField)
		if !ok || key == "" {
			continue
		}
		readonly := isReadonly(f.StructField)
		var inner *look
		if !readonly {
			inner = m.lookFor(f.Type)
		}
		o.names = append(o.names, key)
		o.readonly = append(o.readonly, readonly)
		o.inner = append(o.inner, inner)
	}
	o.index()
	return o
}

// topField is a key of a plan's body, with the type of its field and
// whether it is readonly.
type topField struct {
	key      string
	typ      reflect.Type
	readonly bool
}

// topObject makes the object of a plan's body, whose fields' types may hold
// readonly fields and values that cannot be null however deep, and keeps a
// look only where one does. Its readonly keys are there to be refused, as
// they are one level down; the view, which the body is decoded into, leaves
// them out.
func topObject(fields []topField) *object {
	m := objects{}
	top := &object{}
	for _, f := range fields {
		top.names = append(top.names, f.key)
		top.readonly = append(top.readonly, f.readonly)
		var inner *look
		if !f.readonly {
			inner = m.lookFor(f.typ)
		}
		top.inner = append(top.inner, inner)
	}
	top.index()
	// An object is deep when it has a readonly field or a field whose value
	// needs a look: one that may not be null, or that can hold a deep
	// object. A type can hold itself, so this is found by marking until
	// nothing changes.
	deep := map[*object]bool{}
	var needed func(l *look) bool
	needed = func(l *look) bool {
		return l != nil && (l.notNull || l.size >= 0 || l.obj != nil && deep[l.obj] || needed(l.list) || needed(l.dict))
	}
	for changed := true; changed; {
		changed = false
		for _, o := range m {
			if deep[o] {
				continue
			}
			for i := range o.names {
				if o.readonly[i] || needed(o.inner[i]) {
					deep[o], changed = true, true
					break
				}
			}
		}
	}
	var prune func(l *look) *look
	prune = func(l *look) *look {
		if !needed(l) {
			return nil
		}
		if l.obj != nil && !deep[l.obj] {
			l.obj = nil
		}
		l.list, l.dict = prune(l.list), prune(l.dict)
		return l
	}
	for _, o := range append(slices.Collect(maps.Values(m)), top) {
		for i, l := range o.inner {
			o.inner[i] = prune(l)
		}
	}
	return top
}

// match returns the index of o's field that a JSON object's key, written
// as sent, sets: the field of the key itself, else the one whose key is the
// same once folded; and the key as text. It is -1 when there is none. A key
// is unquoted and folded in the scan's own buffers, so it allocates nothing
// unless the key's text is longer than 64 bytes and has an escape, or is
// longer than 64 bytes and could still match a field once folded.
func (s *bodyScan) match(o *object, quoted []byte) (int, []byte) {
	name := quoted[1 : len(quoted)-1]
	if needsUnquote(name) {
		name = appendUnquoted(s.keyBuf[:0], name)
	}
	s.compares++
	if k, ok := o.exact[string(name)]; ok {
		return k, name
	}
	// A rune of up to 4 bytes folds to one of at least 1, so a key longer
	// than 4 times every folded name cannot be one of them.
	if len(name) <= 4*o.longest {
		s.compares++
		if k, ok := o.folded[string(appendFolded(s.foldBuf[:0], name))]; ok {
			return k, name
		}
	}
	return -1, name
}

// needsUnquote reports whether the text of a JSON string, without its
// quotes, is not what it says: it has an escape or is not well-formed UTF-8,
// which encoding/json writes as U+FFFD. (The body is valid JSON, so it has
// no bare quote or control character.)
func needsUnquote(s []byte) bool {
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\\' {
			return true
		}
		if c < utf8.RuneSelf {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if r == utf8.RuneError && size == 1 {
			return true
		}
		i += size
	}
	return false
}

// appendUnquoted appends what the text of a JSON string, without its quotes,
// says, as encoding/json's unquote reads it: each escape, a surrogate pair
// as its rune and a lone surrogate or malformed UTF-8 as U+FFFD.
func appendUnquoted(dst, s []byte) []byte {
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s):
			switch e := s[i+1]; e {
			case 'b':
				dst, i = append(dst, '\b'), i+2
			case 'f':
				dst, i = append(dst, '\f'), i+2
			case 'n':
				dst, i = append(dst, '\n'), i+2
			case 'r':
				dst, i = append(dst, '\r'), i+2
			case 't':
				dst, i = append(dst, '\t'), i+2
			case 'u':
				r := hex4(s[i+2:])
				i += 6
				if utf16.IsSurrogate(r) {
					if i+1 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
						if pair := utf16.DecodeRune(r, hex4(s[i+2:])); pair != unicode.ReplacementChar {
							dst, i = utf8.AppendRune(dst, pair), i+6
							break
						}
					}
					r = unicode.ReplacementChar
				}
				dst = utf8.AppendRune(dst, r)
			default: // " \ /
				dst, i = append(dst, e), i+2
			}
		case c < utf8.RuneSelf:
			dst, i = append(dst, c), i+1
		default:
			r, size := utf8.DecodeRune(s[i:])
			dst, i = utf8.AppendRune(dst, r), i+size
		}
	}
	return dst
}

// hex4 reads the four hex digits at the start of s, which valid JSON has
// after \u; -1 when there are not four.
func hex4(s []byte) rune {
	if len(s) < 4 {
		return -1
	}
	var r rune
	for _, c := range s[:4] {
		switch {
		case '0' <= c && c <= '9':
			c -= '0'
		case 'a' <= c && c <= 'f':
			c -= 'a' - 10
		case 'A' <= c && c <= 'F':
			c -= 'A' - 10
		default:
			return -1
		}
		r = r*16 + rune(c)
	}
	return r
}

// appendFolded appends name folded as encoding/json folds a key: ASCII
// letters in upper case, and every other rune as the smallest rune of its
// fold set, so that two keys fold the same exactly when bytes.EqualFold
// says they are equal.
func appendFolded(dst, name []byte) []byte {
	for i := 0; i < len(name); {
		if c := name[i]; c < utf8.RuneSelf {
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			dst, i = append(dst, c), i+1
			continue
		}
		r, size := utf8.DecodeRune(name[i:])
		for {
			next := unicode.SimpleFold(r)
			if next <= r {
				r = next
				break
			}
			r = next
		}
		dst, i = utf8.AppendRune(dst, r), i+size
	}
	return dst
}

// walk reads data, a JSON object (valid, though encoding/json may have
// refused its types), in one pass. It returns the top-level keys it sent, in
// its order, each once and as the route names it, or nil when it sent none;
// or, when it sends a readonly field anywhere inside it, the answer that the
// route does not take it; or, when it sends null for a value that cannot be
// null, the answer naming each such field: Send a value, not null. It
// allocates the list it returns, and for a key only what match says.
func (p *plan) walk(data []byte) ([]string, error) {
	s := bodyScan{data: data, mapAt: -1}
	if _, err := s.object(0, p.keys, true); err != nil {
		return nil, err
	}
	if s.nulls != nil {
		return nil, wrongType(s.nulls)
	}
	return s.sent, nil
}

// refusedKey is walk's answer when data sends a readonly field, else nil.
func (p *plan) refusedKey(data []byte) error {
	s := bodyScan{data: data, mapAt: -1}
	_, err := s.object(0, p.keys, true)
	return err
}

// notNull is what a field that was sent null and cannot be says.
const notNull = "Send a value, not null."

// bodyScan is one walk over a body.
type bodyScan struct {
	data []byte
	sent []string
	seen uint64 // the first 64 of the top keys sent; more for the others
	more []bool
	// match's scratch, for a key unquoted and for it folded.
	keyBuf, foldBuf [64]byte
	// path[:depth] is the keys of the objects the scan is in, from the top,
	// as the route names them; a list's index is not in it. mapAt is the
	// depth where the scan entered a map's values, -1 outside one: a
	// field's path ends at the map's field, as a type error's does, since a
	// key may hold dots. nulls is each path sent null that cannot be.
	path  [maxNesting]string
	depth int // how much of path is in use
	mapAt int
	nulls map[string]string
	// pushes counts the keys put in path, and setAt[d] is the count when
	// path[d] was last set, so recorded can tell which of the path's keys
	// changed since the last field recorded; it is made at the first.
	pushes   int
	setAt    [maxNesting]int
	recorded *recordedPaths
	// compares counts each look of a key among the fields, a map lookup or
	// a comparison with one field, which TestScanWideStructCost and
	// TestScanDeepBodyCost bound.
	compares int
	// joins counts the paths joined for wrong, which TestScanNullsCost
	// bounds.
	joins int
}

// object reads the object at i as o's and returns where it ends. With top it
// records the keys sent.
func (s *bodyScan) object(i int, o *object, top bool) (int, error) {
	i = s.space(i + 1)
	if s.data[i] == '}' {
		return i + 1, nil
	}
	for {
		end := s.str(i)
		k, name := s.match(o, s.data[i:end])
		if k >= 0 && o.readonly[k] {
			return 0, unknown(string(name))
		}
		if top && k >= 0 {
			s.record(o, k)
		}
		i = s.space(s.space(end) + 1) // past the colon
		var err error
		if k >= 0 && o.inner[k] != nil {
			s.path[s.depth] = o.names[k]
			s.pushes++
			s.setAt[s.depth] = s.pushes
			s.depth++
			i, err = s.value(i, o.inner[k])
			s.depth--
		} else {
			i = s.skip(i)
		}
		if err != nil {
			return 0, err
		}
		if i = s.space(i); s.data[i] == '}' {
			return i + 1, nil
		}
		i = s.space(i + 1) // past the comma
	}
}

func (s *bodyScan) record(o *object, k int) {
	if bit := uint64(1) << k; k < 64 {
		if s.seen&bit != 0 {
			return
		}
		s.seen |= bit
	} else {
		if s.more == nil {
			s.more = make([]bool, len(o.names))
		}
		if s.more[k] {
			return
		}
		s.more[k] = true
	}
	if s.sent == nil {
		s.sent = make([]string, 0, len(o.names))
	}
	s.sent = append(s.sent, o.names[k])
}

// value reads the value at i as l says, and returns where it ends.
func (s *bodyScan) value(i int, l *look) (int, error) {
	var err error
	switch c := s.data[i]; {
	case c == 'n': // null
		if l.notNull {
			s.null()
		}
		return i + len("null"), nil
	case l.obj != nil && c == '{':
		return s.object(i, l.obj, false)
	case (l.list != nil || l.size >= 0) && c == '[':
		n := 0
		defer func() {
			if l.size >= 0 && n != l.size && err == nil {
				s.wrong(fmt.Sprintf("Send a list of %d %s.", l.size, map[bool]string{true: "value", false: "values"}[l.size == 1]))
			}
		}()
		if i = s.space(i + 1); s.data[i] == ']' {
			return i + 1, nil
		}
		for {
			n++
			if l.list != nil {
				i, err = s.value(i, l.list)
			} else {
				i = s.skip(i)
			}
			if err != nil {
				return 0, err
			}
			if i = s.space(i); s.data[i] == ']' {
				return i + 1, nil
			}
			i = s.space(i + 1)
		}
	case l.dict != nil && c == '{':
		was := s.mapAt
		if was < 0 {
			s.mapAt = s.depth
		}
		i, err = s.values(i, l.dict)
		s.mapAt = was
		return i, err
	}
	return s.skip(i), nil
}

// values reads the object at i as a map whose values l says how to read.
func (s *bodyScan) values(i int, l *look) (int, error) {
	var err error
	if i = s.space(i + 1); s.data[i] == '}' {
		return i + 1, nil
	}
	for {
		i = s.space(s.space(s.str(i)) + 1)
		if i, err = s.value(i, l); err != nil {
			return 0, err
		}
		if i = s.space(i); s.data[i] == '}' {
			return i + 1, nil
		}
		i = s.space(i + 1)
	}
}

// null records that the value at the scan's path was sent null and cannot be.
func (s *bodyScan) null() { s.wrong(notNull) }

// wrong records says for the field at the scan's path, unless a field was
// recorded there already: the path is joined only for the first.
func (s *bodyScan) wrong(says string) {
	n := s.depth
	if s.mapAt >= 0 {
		n = s.mapAt
	}
	if s.recorded == nil {
		s.recorded = &recordedPaths{}
	}
	at := s.recorded.find(s, n)
	if at.done {
		return
	}
	at.done = true
	if s.nulls == nil {
		s.nulls = map[string]string{}
	}
	s.joins++
	if key := strings.Join(s.path[:n], "."); s.nulls[key] == "" {
		s.nulls[key] = says
	}
}

// recordedPaths is a tree of the paths of the fields wrong recorded, a node a
// key, and the nodes of the path it was last asked about: a key set before
// that, and so the same now, is not looked up again. A body of nulls at one
// path, or alternating between a few, costs a look or two each, and an
// allocation only for a path not seen before.
type recordedPaths struct {
	root   pathNode
	at     [maxNesting]*pathNode // at[d] is the node of path[:d+1], for d < n
	n      int
	pushes int // the scan's pushes when at was brought up to date
	looks  int // the keys looked up, which TestScanNullsCost bounds
}

type pathNode struct {
	next map[string]*pathNode
	done bool
}

// find is the node of the scan's path[:n].
func (r *recordedPaths) find(s *bodyScan, n int) *pathNode {
	// The keys set since at was made are the deepest ones, since a key is set
	// after the keys of the objects it is in.
	d := min(n, r.n)
	for d > 0 && s.setAt[d-1] > r.pushes {
		d--
	}
	for ; d < n; d++ {
		parent := &r.root
		if d > 0 {
			parent = r.at[d-1]
		}
		r.looks++
		node := parent.next[s.path[d]]
		if node == nil {
			if parent.next == nil {
				parent.next = map[string]*pathNode{}
			}
			node = &pathNode{}
			parent.next[s.path[d]] = node
		}
		r.at[d] = node
	}
	r.n, r.pushes = n, s.pushes
	if n == 0 {
		return &r.root
	}
	return r.at[n-1]
}

// skip returns where the value at i ends.
func (s *bodyScan) skip(i int) int {
	switch s.data[i] {
	case '"':
		return s.str(i)
	case '{', '[':
		depth := 0
		for ; ; i++ {
			switch s.data[i] {
			case '"':
				i = s.str(i) - 1
			case '{', '[':
				depth++
			case '}', ']':
				if depth--; depth == 0 {
					return i + 1
				}
			}
		}
	}
	for ; i < len(s.data); i++ { // a number, true, false or null
		switch s.data[i] {
		case ',', '}', ']', ' ', '\t', '\r', '\n':
			return i
		}
	}
	return i
}

// str returns where the string at i ends, after its closing quote.
func (s *bodyScan) str(i int) int {
	for i++; ; i++ {
		switch s.data[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
}

func (s *bodyScan) space(i int) int {
	for i < len(s.data) && (s.data[i] == ' ' || s.data[i] == '\t' || s.data[i] == '\r' || s.data[i] == '\n') {
		i++
	}
	return i
}
