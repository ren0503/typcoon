//go:build integration

package tenant_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/pagination"
	"github.com/ren0503/tycoon/internal/tenant"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestTenantCRUDAndSoftDelete(t *testing.T) {
	h := testutil.StartPostgres(t)
	h.TruncateAll(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc := tenant.New(h.Pool)
	system := testutil.SystemPrincipal()

	created, err := svc.Create(ctx, system, tenant.CreateInput{Name: "Acme Cruises"})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if created.ID == uuid.Nil || created.Name != "Acme Cruises" {
		t.Fatalf("unexpected tenant: %+v", created)
	}

	principal := testutil.Principal(created.ID)
	got, err := svc.Get(ctx, principal, created.ID)
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("get tenant id = %s, want %s", got.ID, created.ID)
	}

	page, err := svc.List(ctx, principal, pagination.Cursor{}, 20)
	if err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("list returned %d tenants, want 1", len(page.Items))
	}

	name := "Acme Cruises Ltd"
	updated, err := svc.Update(ctx, principal, created.ID, tenant.UpdateInput{Name: &name})
	if err != nil {
		t.Fatalf("update tenant: %v", err)
	}
	if updated.Name != name {
		t.Fatalf("updated name = %q, want %q", updated.Name, name)
	}

	if err := svc.Delete(ctx, principal, created.ID); err != nil {
		t.Fatalf("delete tenant: %v", err)
	}
	if _, err := svc.Get(ctx, principal, created.ID); err == nil {
		t.Fatal("expected soft-deleted tenant to be invisible")
	} else {
		testutil.RequireErrorCode(t, err, "not_found")
	}

	page, err = svc.List(ctx, principal, pagination.Cursor{}, 20)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("list after delete returned %d tenants, want 0", len(page.Items))
	}
}
