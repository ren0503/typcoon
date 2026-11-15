# Baseline Snapshot

Captured: 2026-11-15T04:00:58Z
Commit: 18e99a458678aa0bd32e50818ef36f6867590afc
Branch: master
Working tree (short):
?? docs/style.md
?? specs/006-go-style-compliance/

Go: go version go1.26.8 linux/amd64
golangci-lint: golangci-lint has version 2.14.0 built with go1.27.0 from 114493f9 on 2026-09-24T11:07:15Z
Docker: unavailable

## Baseline Validation Results

- `gofmt -l ./cmd ./internal`: no files listed (exit 0)
- `golangci-lint fmt --diff ./...`: no diff (exit 0)
- `go vet ./...`: no findings (exit 0)
- `golangci-lint run ./...`: 0 issues (exit 0)
- `go test ./...`: all packages pass (exit 0)
- `go build ./cmd/server ./cmd/migrate`: success (exit 0)
- `go test -tags=integration ./...`: not run; Docker daemon unavailable
- Integration compile path: `go test -tags=integration -run '^$' ./...` (pass, exit 0)

Baseline caveats: `docs/style.md` and `specs/006-go-style-compliance/` are untracked at baseline; no tracked source changes exist.

## Immutable Artifact Hashes
```text
f065ca932fda4ca73ee95d92c43e92988af029b5706ccf1bc32510e8ed78a9af  internal/infrastructure/persistence/postgres/sqlc/ai_feedback.sql.go
67393c18e5717ebb98017d5a178e082907556298761ec7b8418f9aaabb157b33  internal/infrastructure/persistence/postgres/sqlc/auth.sql.go
3b49e1838335d36f370c36e1f183559e908279996ea7b2a45733a04520ccf5f3  internal/infrastructure/persistence/postgres/sqlc/booking.sql.go
168c60a88114d8f895d44dda08e188e26363d68c2d5894e471142ec88d5ca4be  internal/infrastructure/persistence/postgres/sqlc/db.go
2ab38060e756351b79d125c33282d39f1ff5d238f5e8b4b0c9e991e44a9d2ce6  internal/infrastructure/persistence/postgres/sqlc/guest.sql.go
b412642377d09566350ecdccf1483e68dc9abfff8e7f5d139dfe82a5d1894919  internal/infrastructure/persistence/postgres/sqlc/health.sql.go
c364a6e420640bacca03cbfcbbfd076b545c9f08d0f7fc353c245951493cdf2d  internal/infrastructure/persistence/postgres/sqlc/models.go
c7165639f8f58ae0acad6203b920d1f8fa2e12c45a8f33d1ad790b6f339a4763  internal/infrastructure/persistence/postgres/sqlc/parity_integration_test.go
1291f060da1b2d2b27b082d7b118a136aa236f6d5f7f3c40b62bf5a109de3ef2  internal/infrastructure/persistence/postgres/sqlc/partner.sql.go
39dd052a1d8b2c2076b25112c9ece53b4ef774d6cec5d63e778fea719600789f  internal/infrastructure/persistence/postgres/sqlc/querier.go
4a9eb7d54af6aacac16af7c64ad329acb26822f2b3b5437e449180a2bb5d9b3b  internal/infrastructure/persistence/postgres/sqlc/ship.sql.go
7c8b5c65d1ce4758238f95136a0a55ab3039d5073659bccee610cbb1b5a38e5c  internal/infrastructure/persistence/postgres/sqlc/ship_imports.sql.go
bd0163fc48fb4336e051a76024c4e8e59e092a9632cc6b8cb4b38e6dfcb48173  internal/infrastructure/persistence/postgres/sqlc/staff.sql.go
a510c913728f4a0bb9a1f0a5484f8088b0de11a1e07c4ccc0d09ef65cfa06b62  internal/infrastructure/persistence/postgres/sqlc/sync.sql.go
5089109dd58f5f1355ab58fafa62566680961704bead48562ed63c4b9a02f63e  internal/infrastructure/persistence/postgres/sqlc/tenant.sql.go
4f93ad5749949070d91b5395e82f6efbcfd11727c78f7000f3d7a7be60bd4648  internal/infrastructure/persistence/postgres/sqlc/voyage.sql.go
96c67926a259f01d130ecad30560ed3d60c38d2e4ee849ab58e17ab082af91fa  db/migrations/00001_init.sql
1a1f8b5738072b057d128162ca8a3b829987b5ea4b045caf5fc9deba23b62c3e  db/migrations/00002_business_schema.sql
eaf8a6c617c4c1c391f8ca25984b9b82ce3e32343c275c0aa7fb715517486a66  db/migrations/00003_firebase_sso_auth.sql
37f5a56260368ccb0ebcd64660e0f362d7b29f0152c441474023a7222fa34bad  db/migrations/00004_ship_structure_imports.sql
4b7609f03985e6654b3e9aa5a2d7eeebc266423a09bc6e4c2e3625227c690bfb  docs/schema.sql
e79438f8e9045e0e48314a0552f8e8923a4d748f3a6d37494389e6dabb31df8d  openapi/openapi.yaml
```

## Behavior Contract Evidence

- `TestOpenAPISpecIsValid`: pass (OpenAPI document parses and validates).
- `TestRouteInventoryExcludesInternalTables`: pass (route inventory excludes internal tables).
- `TestBusinessRouteTreeBuilds`: pass (business router registers without conflict).
- Immutable artifact hashes: captured above and unchanged through the refactor.
- Schema/migration parity and trigger suites are integration-gated; they compile but were not executed because Docker is unavailable.

## Batch Validation Dry Run

Canonical sequence: `make validate` (fmt-check, vet, lint, test, build, integration compile-check).
Dry run against the untouched tree: all Docker-independent gates passed; the integration compile-check passed; runtime integration suites were skipped because Docker is unavailable.
