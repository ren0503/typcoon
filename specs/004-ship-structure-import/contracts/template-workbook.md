# Template and Validation Contract

The downloadable template and the validator MUST be generated/derived from the same
rules so they cannot drift (FR-012/FR-013). This document is the reference both
implementations and tests use.

## Workbook limits and file rules

| Rule | Value |
|---|---|
| Format | `.xlsx` only (ZIP/OOXML) |
| Max upload size | 10 MiB compressed |
| Max archive expansion | 64 MiB total uncompressed; 32 MiB per entry |
| Max data rows per required sheet | 5,000 non-blank rows |
| Sheets | `Decks`, `Areas`, `Rooms` required; extra sheets ignored |
| Macros | Rejected if `xl/vbaProject.bin` is present |
| Encrypted / OLE / corrupt | Rejected (no ZIP signature or parse failure) |
| Header row | Row 1 on each required sheet |
| Column matching | Required columns matched case-insensitively after trimming; extra columns ignored |
| Blank data rows | Skipped and not counted |
| Value trimming | Names/numbers/strings are trimmed before storage |

## Decks sheet

| Column | Required | Type | Rule |
|---|---|---|---|
| `level` | yes | whole number | `0 ≤ level ≤ 2147483647`; unique within workbook and against existing non-deleted decks of the ship |
| `name` | yes | text | non-blank after trimming |

Example rows:

| level | name |
|---|---|
| 1 | Main Deck |
| 2 | Upper Deck |

## Areas sheet

| Column | Required | Type | Rule |
|---|---|---|---|
| `deck_level` | yes | whole number | Must equal a valid `level` in this workbook's Decks sheet |
| `name` | yes | text | non-blank after trimming |
| `area_type` | yes | enum | one of the values below |
| `capacity` | no | whole number | `≥ 0` when present; blank → NULL |
| `layout` | no | JSON | syntactically valid JSON when present; blank → NULL |

`area_type` values (must match the service/DB check):

```text
restaurant, lobby, open_deck, theatre, shop, corridor, stairwell, spa, pool, other
```

Example rows:

| deck_level | name | area_type | capacity | layout |
|---|---|---|---|---|
| 1 | Main Lobby | lobby | 40 | {"x":0,"y":0,"w":10,"h":8} |
| 1 | Sunset Restaurant | restaurant | 120 | |

## Rooms sheet

| Column | Required | Type | Rule |
|---|---|---|---|
| `deck_level` | yes | whole number | Must equal a valid `level` in this workbook's Decks sheet |
| `number` | yes | text | non-blank after trimming; unique within workbook and against existing non-deleted rooms of the ship |
| `room_type` | yes | enum | one of the values below |
| `capacity` | yes | whole number | `1 ≤ capacity ≤ 32767` (smallint, > 0) |
| `is_bookable` | no | boolean | `true/false/yes/no/1/0` case-insensitive; blank → `true` |
| `layout` | no | JSON | syntactically valid JSON when present; blank → NULL |

`room_type` values (must match the service/DB check):

```text
inside, oceanview, balcony, suite, crew, other
```

Example rows:

| deck_level | number | room_type | capacity | is_bookable | layout |
|---|---|---|---|---|---|
| 2 | 201 | balcony | 2 | true | |
| 2 | 202 | suite | 4 | TRUE | {"x":12,"y":0,"w":6,"h":5} |

## Reference sheet

The generated template includes an extra `Reference` sheet (ignored by validation)
with:

- one row of descriptions for every required column;
- the full `area_type` and `room_type` value lists;
- the deck `level` rule (`≥ 0`, unique per ship among non-deleted decks);
- the capacity rules (Area optional and `≥ 0`; Room required and `> 0`);
- a plain-language explanation that `deck_level` in Areas/Rooms is the `level` of a
  row in the Decks sheet, not a database id;
- the accepted `is_bookable` forms and the default;
- the `.xlsx`-only, 10 MiB, 5,000-row limits.

## Validation findings

Findings are returned in `errors` with `sheet`, `row` (1-based; 0 for file/sheet
level), `column` (canonical name or empty), `code`, and a safe `message`.

| Code | Meaning |
|---|---|
| `file_invalid` | Not a readable OOXML workbook / corrupt content |
| `file_too_large` | Compressed upload exceeds 10 MiB |
| `unsupported_format` | Extension or content is not `.xlsx` |
| `macro_enabled` | `xl/vbaProject.bin` present |
| `archive_too_large` | Archive expansion exceeds the safety limits |
| `missing_sheet` | A required sheet is absent |
| `duplicate_sheet` | Two sheets normalize to the same required name |
| `missing_column` | A required header is absent |
| `duplicate_column` | A required header appears more than once |
| `too_many_rows` | More than 5,000 non-blank data rows in a sheet |
| `required` | A required cell is blank/whitespace |
| `invalid_type` | Value is not the required whole number/text/type |
| `invalid_enum` | Value is not an allowed `area_type`/`room_type` |
| `invalid_capacity` | Area capacity `< 0`, Room capacity `≤ 0` or `> 32767`, or non-integer |
| `invalid_json` | `layout` is present but not syntactically valid JSON |
| `invalid_boolean` | `is_bookable` is not a recognized boolean form |
| `duplicate` | Deck level or Room number duplicated within the workbook |
| `conflict` | Deck level or Room number conflicts with an existing non-deleted record |
| `invalid_reference` | Area/Room `deck_level` does not resolve to a valid Deck row in this workbook |
| `missing_deck_records` | Areas/Rooms exist but the Decks sheet has no data |
| `no_importable_records` | The workbook contains no non-blank data rows |

## Confirmability

An import is `ready` only when there is at least one data row and no findings for any
entity. Any finding anywhere keeps the import `failed` and `can_confirm = false`
(FR-034, SC-005). Duplicate keys and existing-record conflicts mark every affected
row invalid rather than silently skipping it.

Areas and Rooms may only reference Decks declared in the same workbook. Importing
Areas/Rooms into pre-existing decks is not part of this feature; a workbook deck level
that conflicts with an existing non-deleted deck makes its Areas/Rooms references
invalid as well.

## Template examples must validate

A test downloads the generated template, uploads it unchanged for a fresh
non-deleted ship, and expects `ready` with the example counts. Any change to an enum,
column, limit, or default must update the template, this document, the validator, and
that test together.
