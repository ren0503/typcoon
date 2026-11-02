package app_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type listedPackage struct {
	ImportPath string
	Imports    []string
}

var modules = map[string]bool{
	"tenant": true, "ship": true, "voyage": true, "staff": true, "partner": true,
	"booking": true, "guest": true, "sync": true, "ai": true, "feedback": true,
	"auth": true, "audit": true,
}

var infra = map[string]bool{
	"app": true, "apperrors": true, "config": true, "database": true, "httpserver": true,
	"httpx": true, "pagination": true, "testutil": true, "ownership": true, "awsconfig": true,
	"health": true, "logging": true,
}

func TestModuleBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "./...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	graph := map[string][]string{}
	for dec.More() {
		var pkg listedPackage
		if err := dec.Decode(&pkg); err != nil {
			t.Fatalf("decode package: %v", err)
		}
		graph[pkg.ImportPath] = pkg.Imports
	}

	const prefix = "github.com/ren0503/tycoon/internal/"
	for pkg, imports := range graph {
		owner := owningModule(pkg)
		if owner == "" {
			continue
		}
		for _, imported := range imports {
			if !strings.HasPrefix(imported, prefix) {
				continue
			}
			target := strings.TrimPrefix(imported, prefix)
			parts := strings.Split(target, "/")
			if len(parts) != 1 || infra[parts[0]] {
				continue
			}
			if !modules[parts[0]] {
				t.Errorf("%s imports unowned internal package %s", pkg, imported)
				continue
			}
			if parts[0] != owner {
				t.Logf("%s imports public surface of module %s", pkg, parts[0])
			}
		}
	}

	assertAcyclic(t, graph, prefix)
}

func owningModule(pkg string) string {
	const prefix = "github.com/ren0503/tycoon/internal/"
	if !strings.HasPrefix(pkg, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(pkg, prefix), "/")
	if len(parts) == 0 || !modules[parts[0]] {
		return ""
	}
	return parts[0]
}

func assertAcyclic(t *testing.T, graph map[string][]string, prefix string) {
	t.Helper()
	state := map[string]int{}
	var visit func(string)
	visit = func(pkg string) {
		if state[pkg] == 1 {
			t.Fatalf("import cycle detected at %s", pkg)
		}
		if state[pkg] == 2 {
			return
		}
		state[pkg] = 1
		for _, dep := range graph[pkg] {
			if strings.HasPrefix(dep, prefix) {
				visit(dep)
			}
		}
		state[pkg] = 2
	}
	for pkg := range graph {
		if strings.HasPrefix(pkg, prefix) {
			visit(pkg)
		}
	}
}
