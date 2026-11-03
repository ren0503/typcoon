package ownership

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

var createTableRE = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z_][a-z0-9_]*)`)

func TestOwnershipCoversAllTables(t *testing.T) {
	f, err := os.Open("../../docs/schema.sql")
	if err != nil {
		t.Fatalf("open schema: %v", err)
	}
	defer func() { _ = f.Close() }()

	found := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := createTableRE.FindStringSubmatch(strings.ToLower(scanner.Text())); m != nil {
			found[m[1]] = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan schema: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("no tables parsed from docs/schema.sql")
	}

	for table := range found {
		if _, ok := TableOwner[table]; !ok {
			t.Errorf("table %q has no owning module", table)
		}
	}
	for table := range TableOwner {
		if !found[table] {
			t.Errorf("ownership lists unknown table %q", table)
		}
	}
	if len(found) != len(TableOwner) {
		t.Fatalf("schema has %d tables, ownership maps %d", len(found), len(TableOwner))
	}

	seenModules := make(map[Module]bool)
	for _, m := range Modules {
		if seenModules[m] {
			t.Errorf("module %q listed twice", m)
		}
		seenModules[m] = true
	}
	if len(seenModules) != 12 {
		t.Fatalf("expected 12 business modules, got %d", len(seenModules))
	}
	for table, m := range TableOwner {
		if !seenModules[m] {
			t.Errorf("table %q assigned to unknown module %q", table, m)
		}
	}
}
