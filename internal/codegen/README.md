# Walens Code Generation

This directory contains the Walens-specific Go-Jet generation tooling.

## Architecture

The generation pipeline consists of:

1. **Migrations** (`internal/db/migrations/`) - SQL migrations run against a temp SQLite DB
2. **Go-Jet Generation via API** - Uses `sqlite.GenerateDB()` with custom template to generate models directly

The generation is now fully API-driven, not CLI-based. The `cmd/walensgen` tool uses:
- `sqlite.GenerateDB()` to invoke generation
- `template.Default(dialect)` with `UseSchema/UseModel/UseTable/UseField` for customization
- Walens codegen library helpers (`BuildWalensTemplate`, `ClassifyColumnForGenerator`, `createDocTag`) for type mapping

## Custom Types

Generated models use these custom wrapper types:

| Column Pattern | Go Type | DB Storage | Notes |
|--------------|---------|------------|-------|
| `id`, `*_id` (internal) | `dbtypes.UUID` | TEXT | UUIDv7 for internal IDs |
| explicit timestamp columns like `created_at`, `updated_at`, `run_after`, `started_at`, `finished_at` | `dbtypes.UnixMilliTime` | INTEGER ms | RFC3339 in JSON |
| `*_ms` (duration columns) | `dbtypes.UnixMilliDuration` | INTEGER ms | Integer ms in JSON |
| `is_*` | `dbtypes.BoolInt` | INTEGER 0/1 | Boolean in JSON |
| `json_*`, `params`, `metadata` | `dbtypes.RawJSON` | TEXT | Raw JSON in JSON |
| External IDs (`*_identifier`) | `string` | TEXT | NOT converted to UUID |

## Type Classification

Classification is done using column name patterns and explicit sets in `metadata.go`:

- `IsUUIDColumn` - matches `id` or `*_id` but NOT `*_identifier`
- `IsUnixMilliTimestampColumn` - matches known timestamp column names only
- `IsDurationColumn` - matches `*_ms` columns in known set only
- `IsBoolIntColumn` - matches `is_*` columns
- `IsRawJSONColumn` - matches `json_*` or known JSON column names

## Running Generation

```bash
# Build the codegen tool
go build -o walensgen ./cmd/walensgen

# Generate (uses temp DB by default)
./walensgen generate

# With custom paths
./walensgen generate -out ./internal/db/generated -migrations ./internal/db/migrations
```

## Generator API Customization

The Walens template is built in `internal/codegen.BuildWalensTemplate()` and consumed by `cmd/walensgen`:

```go
tmpl := template.Default(sqlite.Dialect)
tmpl = tmpl.UseSchema(func(schemaMetaData metadata.Schema) template.Schema {
    schemaTemplate := template.DefaultSchema(schemaMetaData)
    schemaTemplate.Model = schemaTemplate.Model.UseTable(func(tableMetaData metadata.Table) template.TableModel {
        tableModel := template.DefaultTableModel(tableMetaData)
        tableModel = tableModel.UseField(func(columnMetaData metadata.Column) template.TableModelField {
            field := template.DefaultTableModelField(columnMetaData)
            // Custom type, tags, etc.
            return field
        })
        return tableModel
    })
    return schemaTemplate
})
```

## Adding a New Table

When adding a new table:

1. Add migration to `internal/db/migrations/`
2. Add column documentation to `internal/codegen/metadata.go` in `WalensSchemaDocs`
3. Run `./walensgen generate`
4. Verify generated model uses correct types
5. Add tests for any new codegen helpers
