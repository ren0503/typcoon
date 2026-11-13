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

const modulePrefix = "github.com/ren0503/tycoon/internal/"

type layer int

const (
	layerUnknown layer = iota
	layerApplication
	layerInfrastructure
	layerTransport
	layerPlatform
	layerApp
)

var forbiddenApplicationImports = []string{
	"net/http",
	"database/sql",
	"github.com/go-chi/",
	"github.com/jackc/pgx/",
	"github.com/jackc/pgerrcode",
	"github.com/aws/aws-sdk-go-v2",
	"github.com/redis/go-redis",
	"firebase.google.com/",
	"github.com/testcontainers/",
	"github.com/pressly/goose",
}

func TestLayerBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "github.com/ren0503/tycoon/...")
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

	for pkg, imports := range graph {
		if strings.HasPrefix(pkg, modulePrefix) {
			if importer := classify(pkg); importer == layerUnknown {
				t.Errorf("%s is outside the application/infrastructure/transport/platform/app layers", pkg)
				continue
			}
		}
		importer := classify(pkg)
		if importer == layerUnknown {
			continue
		}
		for _, imported := range imports {
			if importer == layerApplication {
				if forbidden := forbiddenApplicationImport(imported); forbidden != "" {
					t.Errorf("%s (application) imports forbidden package %s", pkg, forbidden)
					continue
				}
			}
			target := classify(imported)
			if target == layerUnknown {
				continue
			}
			if !allowedImport(importer, pkg, target, imported) {
				t.Errorf("%s (%s) must not import %s (%s)", pkg, layerName(importer), imported, layerName(target))
			}
		}
	}

	assertAcyclic(t, graph)
}

func classify(pkg string) layer {
	if pkg == modulePrefix+"app" || strings.HasPrefix(pkg, modulePrefix+"app/") {
		return layerApp
	}
	switch {
	case strings.HasPrefix(pkg, modulePrefix+"application/"):
		return layerApplication
	case strings.HasPrefix(pkg, modulePrefix+"infrastructure/"):
		return layerInfrastructure
	case strings.HasPrefix(pkg, modulePrefix+"transport/"):
		return layerTransport
	case strings.HasPrefix(pkg, modulePrefix+"platform/"):
		return layerPlatform
	}
	return layerUnknown
}

func allowedImport(importer layer, pkg string, target layer, imported string) bool {
	switch importer {
	case layerApplication:
		return target == layerApplication || target == layerPlatform
	case layerInfrastructure:
		return target == layerApplication || target == layerInfrastructure || target == layerPlatform
	case layerTransport:
		return target == layerApplication || target == layerTransport || target == layerPlatform
	case layerPlatform:
		if strings.HasPrefix(pkg, modulePrefix+"platform/testutil") {
			return target == layerApplication || target == layerInfrastructure || target == layerPlatform
		}
		return target == layerPlatform
	case layerApp:
		return true
	}
	return true
}

func forbiddenApplicationImport(imported string) string {
	for _, prefix := range forbiddenApplicationImports {
		if imported == prefix || strings.HasPrefix(imported, prefix) {
			return prefix
		}
	}
	return ""
}

func layerName(l layer) string {
	switch l {
	case layerApplication:
		return "application"
	case layerInfrastructure:
		return "infrastructure"
	case layerTransport:
		return "transport"
	case layerPlatform:
		return "platform"
	case layerApp:
		return "app"
	}
	return "unknown"
}

func assertAcyclic(t *testing.T, graph map[string][]string) {
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
			if strings.HasPrefix(dep, modulePrefix) {
				visit(dep)
			}
		}
		state[pkg] = 2
	}
	for pkg := range graph {
		if strings.HasPrefix(pkg, modulePrefix) {
			visit(pkg)
		}
	}
}
