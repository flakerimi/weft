package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strconv"
)

const (
	mailPackage       = "github.com/flakerimi/weft/warp/mail"
	middlewarePackage = "github.com/flakerimi/weft/warp/middleware"
)

// errCannotAddMail is what weft upgrade says of an app/modules.go that lists
// the login without mail and that it cannot edit.
var errCannotAddMail = errors.New(`app/modules.go lists auth.Module() without mail.Module(), which the login needs from Weft 0.2.0, and weft upgrade could not add it; add the line mail.Module(), before auth.Module() in Modules(), and the import "github.com/flakerimi/weft/warp/mail", then run weft gen.`)

// addMailModule lists mail.Module() just before auth.Module() in the
// []warp.Module{...} that Modules() returns, with the import, when the file
// imports warp/auth and not warp/mail, as an app made before 0.2.0 does; it
// reports whether it changed src. The edit is made in the bytes, as weft
// scaffold's is, and the file formatted, so every other line and comment is
// kept. A file that imports the login without mail and that it cannot edit
// (one that does not parse, a list built by hand, another package named
// mail) is errCannotAddMail.
func addMailModule(src []byte) ([]byte, bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app/modules.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		if bytes.Contains(src, []byte(strconv.Quote(authPackage))) && !bytes.Contains(src, []byte(strconv.Quote(mailPackage))) {
			return nil, false, errCannotAddMail
		}
		return src, false, nil
	}
	local := imported(f)
	if local[authPackage] == "" || local[mailPackage] != "" {
		return src, false, nil
	}
	list := moduleList(f, local[warpPackage])
	if list == nil {
		return nil, false, errCannotAddMail
	}
	for _, name := range local {
		if name == "mail" { // another package by that name: the author's to settle
			return nil, false, errCannotAddMail
		}
	}
	at := slices.IndexFunc(list.Elts, func(e ast.Expr) bool { return isModuleCall(e, local[authPackage]) })
	if at < 0 {
		return nil, false, errCannotAddMail
	}
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	line := func(p token.Pos) int { return fset.Position(p).Line }
	auth := list.Elts[at]
	type edit struct {
		at   int
		text string
	}
	// On a line of its own when auth.Module() is; gofmt indents it.
	call := "mail.Module(),\n"
	if line(auth.Pos()) == line(list.Lbrace) || at > 0 && line(list.Elts[at-1].End()) == line(auth.Pos()) {
		call = "mail.Module(), "
	}
	edits := []edit{{off(auth.Pos()), call}}
	var last *ast.GenDecl
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			last = g
		}
	}
	quoted := strconv.Quote(mailPackage)
	if last == nil || !last.Lparen.IsValid() || line(last.Rparen) == line(last.Lparen) {
		return nil, false, errCannotAddMail // weft create writes one import group; another shape is the author's
	}
	// After the warp import the group holds, so it sorts with Weft's; else at the group's end.
	var after *ast.ImportSpec
	for _, s := range last.Specs {
		spec := s.(*ast.ImportSpec)
		if p, _ := strconv.Unquote(spec.Path.Value); p == authPackage || p == "github.com/flakerimi/weft/warp/data" || p == warpPackage {
			after = spec
		}
	}
	if after != nil {
		end := off(after.End())
		eol := bytes.IndexByte(src[end:], '\n')
		edits = append(edits, edit{end + eol + 1, "\t" + quoted + "\n"})
	} else {
		start := bytes.LastIndexByte(src[:off(last.Rparen)], '\n') + 1
		edits = append(edits, edit{start, "\t" + quoted + "\n"})
	}
	slices.SortFunc(edits, func(a, b edit) int { return b.at - a.at })
	out := slices.Clone(src)
	for _, e := range edits {
		out = slices.Insert(out, e.at, []byte(e.text)...)
	}
	formatted, err := format.Source(out)
	if err != nil {
		return nil, false, errCannotAddMail
	}
	return formatted, true, nil
}

// listsModule reports whether src's Modules() list has the module of the
// package at importPath, as pkg.Module(...).
func listsModule(src []byte, importPath string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "app/modules.go", src, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	local := imported(f)
	list := moduleList(f, local[warpPackage])
	return list != nil && local[importPath] != "" && slices.ContainsFunc(list.Elts, func(e ast.Expr) bool { return isModuleCall(e, local[importPath]) })
}

// imported maps each import path of f to the name the file uses for it.
func imported(f *ast.File) map[string]string {
	local := map[string]string{}
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		n := path.Base(p)
		if imp.Name != nil {
			n = imp.Name.Name
		}
		local[p] = n
	}
	return local
}

// moduleList is the []warp.Module{...} that the function Modules returns, or
// nil.
func moduleList(f *ast.File, warpName string) *ast.CompositeLit {
	var list *ast.CompositeLit
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "Modules" || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				return list == nil
			}
			if lit, ok := ret.Results[0].(*ast.CompositeLit); ok && isModuleList(lit, warpName) {
				list = lit
			}
			return list == nil
		})
	}
	return list
}

// isModuleCall reports whether e is name.Module(...).
func isModuleCall(e ast.Expr, name string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || name == "" {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == name && sel.Sel.Name == "Module"
}
