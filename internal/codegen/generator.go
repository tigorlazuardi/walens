package codegen

import (
	"fmt"

	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/template"
	jetSQLite "github.com/go-jet/jet/v2/sqlite"
)

const dbtypesImportPath = "github.com/walens/walens/internal/dbtypes"

// BuildWalensTemplate constructs the Go-Jet template customization used by Walens.
//
// Customization happens during generation, not through AST postprocessing.
func BuildWalensTemplate(modelPath string) template.Template {
	return template.Default(jetSQLite.Dialect).UseSchema(func(schema metadata.Schema) template.Schema {
		s := template.DefaultSchema(schema)
		s.Path = ""
		s.Model = s.Model.UsePath(modelPath).UseTable(func(table metadata.Table) template.TableModel {
			return template.DefaultTableModel(table).UseField(func(column metadata.Column) template.TableModelField {
				field := template.DefaultTableModelField(column)
				field = field.UseType(createFieldType(table, column))
				field.Tags = buildFieldTags(table, column, field.Tags)
				return field
			})
		})
		s.SQLBuilder.Skip = false
		return s
	})
}

func createFieldType(table metadata.Table, column metadata.Column) template.Type {
	_, wrappedType, needsPointer := ClassifyColumnForGenerator(table.Name, column.Name, column.IsNullable, column.DataType.Name)

	if wrappedType == "" {
		return template.DefaultTableModelField(column).Type
	}

	if wrappedType == "string" || wrappedType == "int64" || wrappedType == "float64" || wrappedType == "[]byte" {
		name := wrappedType
		if needsPointer {
			name = "*" + name
		}
		return template.Type{Name: name}
	}

	name := wrappedType
	if needsPointer {
		name = "*" + name
	}

	return template.Type{
		ImportPath: dbtypesImportPath,
		Name:       name,
	}
}

func buildFieldTags(table metadata.Table, column metadata.Column, existing []string) []string {
	tags := make([]string, 0, len(existing)+2)
	tags = append(tags, existing...)
	if docTag := createDocTag(table, column); docTag != "" {
		tags = append(tags, docTag)
	}
	if !column.IsNullable {
		tags = append(tags, `required:"true"`)
		tags = append(tags, fmt.Sprintf(`json:%q,omitzero`, column.Name))
	} else {
		tags = append(tags, fmt.Sprintf(`json:%q`, column.Name))
	}
	return tags
}

// createDocTag returns the Walens doc tag for a table/column pair.
func createDocTag(table metadata.Table, column metadata.Column) string {
	col := GetColumnDoc(table.Name, column.Name)
	if col.Description == "" {
		return ""
	}
	return DocTag(col.Description)
}
