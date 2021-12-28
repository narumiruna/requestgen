package requestgen

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

type TypeSelector struct {
	Package string
	Member  string
}

func sanitizeImport(ts *TypeSelector) (*TypeSelector, error) {
	buildCtx := build.Default

	cwd, err := os.Getwd()
	if err != nil {
		return ts, err
	}

	bp, err := buildCtx.Import(ts.Package, cwd, build.FindOnly)
	if err != nil {
		return ts, fmt.Errorf("can't find package %q", ts.Package)
	}

	ts.Package = bp.ImportPath
	return ts, nil
}

func ParseTypeSelector(main string) (*TypeSelector, error) {
	var spec TypeSelector

	e, _ := parser.ParseExpr(main)

	if pkg := parseImportPath(e); pkg != "" {
		// e.g. bytes or "encoding/json": a package
		spec.Package = pkg
		return &spec, nil
	}

	if e, ok := e.(*ast.SelectorExpr); ok {
		x := unparen(e.X)

		// Strip off star constructor, if any.
		if star, ok := x.(*ast.StarExpr); ok {
			x = star.X
		}

		if pkg := parseImportPath(x); pkg != "" {
			// package member e.g. "encoding/json".HTMLEscape
			spec.Package = pkg       // e.g. "encoding/json"
			spec.Member = e.Sel.Name // e.g. "HTMLEscape"
			return sanitizeImport(&spec)
		}

		if x, ok := x.(*ast.SelectorExpr); ok {
			// field/method of type e.g. ("encoding/json".Decoder).Decode
			y := unparen(x.X)
			if pkg := parseImportPath(y); pkg != "" {
				spec.Package = pkg       // e.g. "encoding/json"
				spec.Member = x.Sel.Name // e.g. "Decoder"
				return sanitizeImport(&spec)
			}
		}
	}

	return nil, fmt.Errorf("can not parse type selector: %q", main)
}

func unparen(e ast.Expr) ast.Expr { return astutil.Unparen(e) }

func parseImportPath(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name // e.g. bytes

	case *ast.BasicLit:
		if e.Kind == token.STRING {
			pkgname, _ := strconv.Unquote(e.Value)
			return pkgname // e.g. "encoding/json"
		}
	}
	return ""
}
