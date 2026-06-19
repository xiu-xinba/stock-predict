package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
)

type route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

var directMethods = map[string]string{
	"GET":     "GET",
	"POST":    "POST",
	"PUT":     "PUT",
	"DELETE":  "DELETE",
	"PATCH":   "PATCH",
	"HEAD":    "HEAD",
	"OPTIONS": "OPTIONS",
}

var unsupportedGinRouteRegistrations = map[string]bool{
	"Any":          true,
	"Match":        true,
	"Static":       true,
	"StaticFS":     true,
	"StaticFile":   true,
	"StaticFileFS": true,
}

var httpMethodSelectors = map[string]string{
	"MethodGet":     "GET",
	"MethodPost":    "POST",
	"MethodPut":     "PUT",
	"MethodDelete":  "DELETE",
	"MethodPatch":   "PATCH",
	"MethodHead":    "HEAD",
	"MethodOptions": "OPTIONS",
}

func main() {
	routerPath := flag.String("router", "", "path to the Go router source")
	flag.Parse()
	if strings.TrimSpace(*routerPath) == "" {
		fmt.Fprintln(os.Stderr, "--router is required")
		os.Exit(2)
	}
	routes, err := extractRoutes(*routerPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(routes); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func extractRoutes(path string) ([]route, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse router source: %w", err)
	}

	groups := make(map[string]string)
	var routes []route
	var extractionErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if extractionErr != nil {
			return false
		}
		switch value := node.(type) {
		case *ast.AssignStmt:
			if err := recordGroup(value, groups); err != nil {
				extractionErr = err
				return false
			}
		case *ast.CallExpr:
			route, found, err := routeFromCall(value, groups)
			if err != nil {
				extractionErr = err
				return false
			}
			if found {
				routes = append(routes, route)
			}
		}
		return true
	})
	if extractionErr != nil {
		return nil, extractionErr
	}
	if len(routes) == 0 {
		return nil, errors.New("no routes found; refusing to produce an empty contract")
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, nil
}

func recordGroup(assign *ast.AssignStmt, groups map[string]string) error {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return nil
	}
	name, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return nil
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Group" {
		return nil
	}
	if len(call.Args) == 0 {
		return fmt.Errorf("route group %s has no prefix", name.Name)
	}
	prefix, err := stringLiteral(call.Args[0])
	if err != nil {
		return fmt.Errorf("route group %s prefix: %w", name.Name, err)
	}
	if parent, ok := selector.X.(*ast.Ident); ok {
		if parentPrefix, found := groups[parent.Name]; found {
			prefix = joinPaths(parentPrefix, prefix)
		}
	}
	groups[name.Name] = prefix
	return nil
}

func routeFromCall(call *ast.CallExpr, groups map[string]string) (route, bool, error) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return route{}, false, nil
	}
	method, direct := directMethods[selector.Sel.Name]
	isHandle := selector.Sel.Name == "Handle"
	if !direct && !isHandle {
		if receiver, ok := selector.X.(*ast.Ident); ok && groups[receiver.Name] != "" && unsupportedGinRouteRegistrations[selector.Sel.Name] {
			return route{}, false, fmt.Errorf("unsupported Gin route registration %s on group %s", selector.Sel.Name, receiver.Name)
		}
		return route{}, false, nil
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return route{}, false, fmt.Errorf("route call %s has unsupported receiver", selector.Sel.Name)
	}
	prefix, ok := groups[receiver.Name]
	if !ok {
		return route{}, false, fmt.Errorf("route call %s uses unresolved group %s", selector.Sel.Name, receiver.Name)
	}

	pathArg := 0
	if isHandle {
		if len(call.Args) < 2 {
			return route{}, false, errors.New("Handle route requires method and path")
		}
		resolved, err := methodExpression(call.Args[0])
		if err != nil {
			return route{}, false, err
		}
		method = resolved
		pathArg = 1
	} else if len(call.Args) < 1 {
		return route{}, false, fmt.Errorf("%s route requires a path", method)
	}
	routePath, err := stringLiteral(call.Args[pathArg])
	if err != nil {
		return route{}, false, fmt.Errorf("%s route path: %w", method, err)
	}
	return route{Method: method, Path: joinPaths(prefix, routePath)}, true, nil
}

func methodExpression(expr ast.Expr) (string, error) {
	if literal, err := stringLiteral(expr); err == nil {
		return strings.ToUpper(literal), nil
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", errors.New("Handle method must be a string literal or net/http method constant")
	}
	if ident, ok := selector.X.(*ast.Ident); !ok || ident.Name != "http" {
		return "", errors.New("Handle method selector must come from net/http")
	}
	method, ok := httpMethodSelectors[selector.Sel.Name]
	if !ok {
		return "", fmt.Errorf("unsupported net/http method constant %s", selector.Sel.Name)
	}
	return method, nil
}

func stringLiteral(expr ast.Expr) (string, error) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", errors.New("must be a string literal")
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", fmt.Errorf("unquote string: %w", err)
	}
	return value, nil
}

func joinPaths(prefix, path string) string {
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}
