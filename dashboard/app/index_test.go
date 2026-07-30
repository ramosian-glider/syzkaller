// Copyright 2026 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type indexFile struct {
	Indexes []struct {
		Kind       string `yaml:"kind"`
		Ancestor   bool   `yaml:"ancestor"`
		Properties []struct {
			Name      string `yaml:"name"`
			Direction string `yaml:"direction"`
		} `yaml:"properties"`
	} `yaml:"indexes"`
}

type orderInfo struct {
	prop      string
	direction string // "asc" or "desc"
}

type parsedQuery struct {
	file       string
	line       int
	kind       string
	ancestor   bool
	eqFilters  []string
	ineqFilter string
	orders     []orderInfo
}

func TestDatastoreIndexes(t *testing.T) {
	data, err := os.ReadFile("index.yaml")
	require.NoError(t, err)

	var cfg indexFile
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, file, nil, 0)
		require.NoError(t, err)

		ast.Inspect(node, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			for _, query := range extractQueries(fset, file, fn.Body) {
				if !hasMatchingIndex(cfg, query) {
					t.Errorf("%s:%d: query for kind %q (ancestor=%v, eq=%v, ineq=%q, orders=%v) missing from index.yaml",
						query.file, query.line, query.kind, query.ancestor, query.eqFilters, query.ineqFilter, query.orders)
				}
			}
			return true
		})
	}
}

// extractQueries scans a function body for db.NewQuery(...) invocations and extracts all chained or
// variable-assigned query methods (.Ancestor, .Filter, .Order).
func extractQueries(fset *token.FileSet, file string, body *ast.BlockStmt) []parsedQuery {
	var res []parsedQuery
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewQuery" || len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		q := parsedQuery{
			file: file,
			line: fset.Position(call.Pos()).Line,
			kind: strings.Trim(lit.Value, `"`),
		}

		varName := getQueryVarName(call, body)

		var calls []*ast.CallExpr
		ast.Inspect(body, func(cn ast.Node) bool {
			c, ok := cn.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, chainCall := range getFluentChain(c) {
				if isDerivedFrom(chainCall.Fun, call, varName) && !slices.Contains(calls, chainCall) {
					calls = append(calls, chainCall)
				}
			}
			return true
		})

		for _, childCall := range calls {
			methodSel, ok := childCall.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			switch methodSel.Sel.Name {
			case "Ancestor":
				q.ancestor = true
			case "Filter":
				if len(childCall.Args) >= 1 {
					if arg, ok := childCall.Args[0].(*ast.BasicLit); ok && arg.Kind == token.STRING {
						raw := strings.Trim(arg.Value, `"`)
						prop, op := parseFilterProp(raw)
						if prop != "" {
							if op != "=" {
								q.ineqFilter = prop
							} else if !slices.Contains(q.eqFilters, prop) {
								q.eqFilters = append(q.eqFilters, prop)
							}
						}
					}
				}
			case "Order":
				if len(childCall.Args) >= 1 {
					if arg, ok := childCall.Args[0].(*ast.BasicLit); ok && arg.Kind == token.STRING {
						raw := strings.Trim(arg.Value, `"`)
						ord := parseOrderProp(raw)
						if ord.prop != "" && !containsOrder(q.orders, ord.prop) {
							q.orders = append(q.orders, ord)
						}
					}
				}
			}
		}

		if queryNeedsIndex(q) {
			res = append(res, q)
		}
		return true
	})
	return res
}

func containsOrder(orders []orderInfo, prop string) bool {
	for _, o := range orders {
		if o.prop == prop {
			return true
		}
	}
	return false
}

func parseOrderProp(raw string) orderInfo {
	if strings.HasPrefix(raw, "-") {
		return orderInfo{prop: raw[1:], direction: "desc"}
	}
	return orderInfo{prop: strings.TrimPrefix(raw, "+"), direction: "asc"}
}

// getFluentChain unwraps a chained call expression from left-to-right (innermost receiver call to outermost method).
func getFluentChain(expr ast.Expr) []*ast.CallExpr {
	var chain []*ast.CallExpr
	curr := expr
	for curr != nil {
		call, ok := curr.(*ast.CallExpr)
		if !ok {
			break
		}
		chain = append(chain, call)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}
		curr = sel.X
	}
	slices.Reverse(chain)
	return chain
}

// getQueryVarName returns the identifier name if newQueryCall is assigned to a variable (e.g. q := db.NewQuery(...)).
func getQueryVarName(newQueryCall *ast.CallExpr, body *ast.BlockStmt) string {
	var varName string
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			for i, rhs := range stmt.Rhs {
				if isDerivedFrom(rhs, newQueryCall, "") {
					if id, ok := stmt.Lhs[i].(*ast.Ident); ok {
						varName = id.Name
						return false
					}
				}
			}
		}
		return true
	})
	return varName
}

// isDerivedFrom checks whether an expression originates from target (a NewQuery call expression)
// or references varName (the variable storing the query).
func isDerivedFrom(expr ast.Expr, target *ast.CallExpr, varName string) bool {
	for curr := expr; curr != nil; {
		switch node := curr.(type) {
		case *ast.CallExpr:
			if node == target {
				return true
			}
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				curr = sel.X
			} else {
				curr = nil
			}
		case *ast.Ident:
			if varName != "" && node.Name == varName {
				return true
			}
			curr = nil
		case *ast.SelectorExpr:
			curr = node.X
		default:
			curr = nil
		}
	}
	return false
}

// parseFilterProp splits a filter string like "Field >=" into property name ("Field") and operator (">=").
// Multi-character operators are listed first so "<=" and ">=" match before "<" and ">".
func parseFilterProp(raw string) (string, string) {
	for _, op := range []string{"<=", ">=", "!=", "=", "<", ">"} {
		if idx := strings.Index(raw, op); idx != -1 {
			return strings.TrimSpace(raw[:idx]), op
		}
	}
	return strings.TrimSpace(raw), "="
}

// queryNeedsIndex returns true if Datastore requires a composite index for q:
// 1. Ancestor filter combined with Order or inequality filter.
// 2. Multiple Order clauses.
// 3. Filter combined with Order on different properties.
// 4. Inequality filter combined with multiple filters.
func queryNeedsIndex(q parsedQuery) bool {
	hasInequality := q.ineqFilter != ""
	filtersCount := len(q.eqFilters)
	if hasInequality {
		filtersCount++
	}
	ordersCount := len(q.orders)

	if q.ancestor && (ordersCount > 0 || hasInequality) {
		return true
	}
	if ordersCount > 1 || (filtersCount > 0 && ordersCount > 0) {
		return true
	}
	if hasInequality && filtersCount > 1 {
		return true
	}
	return false
}

// hasMatchingIndex returns true if cfg contains a composite index in index.yaml that satisfies q according to Datastore index rules:
// 1. Kind and Ancestor match strictly.
// 2. Leading properties must contain all equality filter properties (in any order).
// 3. Inequality filter property (if any) must immediately follow equality filter properties.
// 4. Sort order properties must follow in exact sequence and match direction (asc/desc).
func hasMatchingIndex(cfg indexFile, q parsedQuery) bool {
	for _, entry := range cfg.Indexes {
		if entry.Kind != q.kind || q.ancestor != entry.Ancestor {
			continue
		}

		props := entry.Properties
		reqEq := len(q.eqFilters)
		if len(props) < reqEq {
			continue
		}

		// 1. Leading index properties must contain all equality filters (in any order).
		eqMatch := true
		for i := 0; i < reqEq; i++ {
			if !slices.Contains(q.eqFilters, props[i].Name) {
				eqMatch = false
				break
			}
		}
		if !eqMatch {
			continue
		}

		currIdx := reqEq
		remainingOrders := q.orders

		// 2. Inequality filter (if any) must immediately follow equality filters.
		if q.ineqFilter != "" {
			if currIdx >= len(props) || props[currIdx].Name != q.ineqFilter {
				continue
			}
			currIdx++
			if len(remainingOrders) > 0 && remainingOrders[0].prop == q.ineqFilter {
				remainingOrders = remainingOrders[1:]
			}
		}

		// 3. Order properties must follow in exact sequence and match sort direction.
		orderMatch := true
		for _, ord := range remainingOrders {
			if currIdx >= len(props) {
				orderMatch = false
				break
			}
			propDir := props[currIdx].Direction
			if propDir == "" {
				propDir = "asc"
			}
			if props[currIdx].Name != ord.prop || propDir != ord.direction {
				orderMatch = false
				break
			}
			currIdx++
		}
		if orderMatch {
			return true
		}
	}
	return false
}
