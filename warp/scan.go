package warp

// The scan of a typed route's body: one pass over bytes encoding/json has
// already decoded into the plan's view, which finds the top-level keys the
// body sent and refuses a readonly key at any depth.
//
// It mirrors encoding/json v1's matching of a key to a field (decode.go,
// object): the field whose key is the key itself, else the first whose key
// folds the same (foldName), the key unquoted as unquoteBytes does. Where
// another decoder could choose another of the folded fields (jsonv2 prefers
// the shallowest), the scan takes all of them into account, so its refusal
// does not depend on the choice. TestScanMatchesEncodingJSON and FuzzScan
// hold it to encoding/json. Should the two part all the same, what holds is
// plan.readonly: every readonly field is put back to zero after decoding, so
// a readonly field is never set, whatever the scan said.

import (
	"cmp"
	"maps"
	"reflect"
	"slices"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// object is what may be sent where a struct is: the json keys of its
// fields in field order, which of them are readonly, and how to look inside
// each one's value, nil where nothing inside it can be readonly. exact finds
// the field of a key as sent; folded the fields whose keys are the same once
// folded, in field order, all of them.
type object struct {
	names    []string
	readonly []bool
	inner    []*look
	exact    map[string]int
	folded   map[string][]int
	single   []int // 0, 1, 2, …: single[k:k+1] is the one field k, without allocating
	longest  int   // the length of the longest folded name, in bytes
}

// index makes o's exact and folded maps from its names.
func (o *object) index() {
	o.exact = make(map[string]int, len(o.names))
	o.folded = make(map[string][]int, len(o.names))
	o.single = make([]int, len(o.names))
	for i, name := range o.names {
		o.exact[name] = i
		o.single[i] = i
		f := string(appendFolded(nil, []byte(name)))
		o.folded[f] = append(o.folded[f], i)
		o.longest = max(o.longest, len(f))
	}
}

// look is how to look inside a JSON value: as a struct's object, as a list
// of values or as an object of values (a map), each value looked at so.
type look struct {
	obj        *object
	list, dict *look
}

// objects makes the objects of the struct types a body reaches, each once,
// so that a type that holds itself ends.
type objects map[reflect.Type]*object

func (m objects) lookFor(t reflect.Type) *look {
	for t.Kind() == reflect.Pointer && !decodesItself(t) {
		t = t.Elem()
	}
	if decodesItself(t) {
		return nil
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		if e := m.lookFor(t.Elem()); e != nil {
			return &look{list: e}
		}
	case reflect.Map:
		if e := m.lookFor(t.Elem()); e != nil {
			return &look{dict: e}
		}
	case reflect.Struct:
		if t != timeType {
			return &look{obj: m.objectOf(t)}
		}
	}
	return nil
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
// readonly fields however deep, and keeps a look only where one does. Its
// readonly keys are there to be refused, as they are one level down; the
// view, which the body is decoded into, leaves them out.
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
	// can hold a deep object. A type can hold itself, so this is found by
	// marking until nothing changes.
	deep := map[*object]bool{}
	reaches := func(l *look) bool {
		for l != nil && l.obj == nil {
			l = cmp.Or(l.list, l.dict)
		}
		return l != nil && deep[l.obj]
	}
	for changed := true; changed; {
		changed = false
		for _, o := range m {
			if deep[o] {
				continue
			}
			for i := range o.names {
				if o.readonly[i] || reaches(o.inner[i]) {
					deep[o], changed = true, true
					break
				}
			}
		}
	}
	for _, o := range append(slices.Collect(maps.Values(m)), top) {
		for i, l := range o.inner {
			if !reaches(l) {
				o.inner[i] = nil
			}
		}
	}
	return top
}

// match returns the fields of o that a JSON object's key, written as sent,
// may set: the field of the key itself, else every field whose key is the
// same once folded, in field order; and the key as text. It is nil when
// there is none. encoding/json (v1) sets the exact field, else the first of
// the folded ones; another decoder may choose another of them (jsonv2 takes
// the shallowest), so the scan refuses a key when any of them is readonly
// and looks inside its value as each of them. A key is unquoted and folded
// in the scan's own buffers, so it allocates nothing unless the key's text
// is longer than 64 bytes and has an escape, or is longer than 64 bytes and
// could still match a field once folded.
func (s *bodyScan) match(o *object, quoted []byte) ([]int, []byte) {
	name := quoted[1 : len(quoted)-1]
	if needsUnquote(name) {
		name = appendUnquoted(s.keyBuf[:0], name)
	}
	s.compares++
	if k, ok := o.exact[string(name)]; ok {
		return o.single[k : k+1], name
	}
	// A rune of up to 4 bytes folds to one of at least 1, so a key longer
	// than 4 times every folded name cannot be one of them.
	if len(name) <= 4*o.longest {
		s.compares++
		if ks, ok := o.folded[string(appendFolded(s.foldBuf[:0], name))]; ok {
			return ks, name
		}
	}
	return nil, name
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
// route does not take it. It allocates the list it returns, and for a key
// only what match says.
func (p *plan) walk(data []byte) ([]string, error) {
	s := bodyScan{data: data}
	if _, err := s.object(0, p.keys, true); err != nil {
		return nil, err
	}
	return s.sent, nil
}

// bodyScan is one walk over a body.
type bodyScan struct {
	data []byte
	sent []string
	seen uint64 // the first 64 of the top keys sent; more for the others
	more []bool
	// match's scratch, for a key unquoted and for it folded.
	keyBuf, foldBuf [64]byte
	// compares counts each look of a key among the fields, a map lookup or
	// a comparison with one field, which TestScanWideStructCost bounds.
	compares int
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
		ks, name := s.match(o, s.data[i:end])
		for _, k := range ks {
			if o.readonly[k] {
				return 0, unknown(string(name))
			}
		}
		if top && len(ks) > 0 {
			s.record(o, ks[0]) // the field encoding/json sets
		}
		i = s.space(s.space(end) + 1) // past the colon
		next, looked := 0, false
		for _, k := range ks {
			if o.inner[k] == nil {
				continue
			}
			var err error
			if next, err = s.value(i, o.inner[k]); err != nil {
				return 0, err
			}
			looked = true
		}
		if !looked {
			next = s.skip(i)
		}
		i = next
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
	case l.obj != nil && c == '{':
		return s.object(i, l.obj, false)
	case l.list != nil && c == '[':
		if i = s.space(i + 1); s.data[i] == ']' {
			return i + 1, nil
		}
		for {
			if i, err = s.value(i, l.list); err != nil {
				return 0, err
			}
			if i = s.space(i); s.data[i] == ']' {
				return i + 1, nil
			}
			i = s.space(i + 1)
		}
	case l.dict != nil && c == '{':
		if i = s.space(i + 1); s.data[i] == '}' {
			return i + 1, nil
		}
		for {
			i = s.space(s.space(s.str(i)) + 1)
			if i, err = s.value(i, l.dict); err != nil {
				return 0, err
			}
			if i = s.space(i); s.data[i] == '}' {
				return i + 1, nil
			}
			i = s.space(i + 1)
		}
	}
	return s.skip(i), nil
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
