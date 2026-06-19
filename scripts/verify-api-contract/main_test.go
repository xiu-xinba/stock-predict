package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRoutesFindsMultilineAndHandleCalls(t *testing.T) {
	routes, err := extractRoutes(filepath.Join("testdata", "router.go"))
	if err != nil {
		t.Fatalf("extract routes: %v", err)
	}

	want := map[string]bool{
		"GET /api/v1/multiline/:code": false,
		"POST /api/v1/handled":        false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("missing route %s in %+v", route, routes)
		}
	}
}

func TestExtractRoutesFailsClosedOnDynamicPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "router.go")
	source := `package fixture
func routes(v1 interface{ GET(string, ...any) }) {
	path := "/dynamic"
	v1.GET(path)
}`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := extractRoutes(path); err == nil {
		t.Fatal("expected dynamic route path to fail closed")
	}
}

func TestExtractRoutesFailsClosedOnUnsupportedGinRegistration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "router.go")
	source := `package fixture
import "github.com/gin-gonic/gin"
func routes(engine *gin.Engine, handler gin.HandlerFunc) {
	v1 := engine.Group("/api/v1")
	v1.GET("/known")
	v1.Any("/all", handler)
}`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := extractRoutes(path); err == nil {
		t.Fatal("expected unsupported Gin route registration to fail closed")
	}
}
