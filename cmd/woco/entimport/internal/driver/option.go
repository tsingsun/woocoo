package driver

// ImportOption allows for managing import configuration using functional options.
type ImportOption func(*ImportOptions)

// WithDSN provides a DSN (data source name) for reading the schema & tables from.
// DSN must include a schema (named-database) in the connection string.
func WithDSN(dsn string) ImportOption {
	return func(i *ImportOptions) {
		i.dsn = dsn
	}
}

// WithSchemaPath provides a DSN (data source name) for reading the schema & tables from.
func WithSchemaPath(path string) ImportOption {
	return func(i *ImportOptions) {
		i.schemaPath = path
	}
}

// WithTables limits the schema import to a set of given tables (by all tables are imported)
func WithTables(tables []string) ImportOption {
	return func(i *ImportOptions) {
		i.tables = tables
	}
}

func WithCaseInt(isCase bool) ImportOption {
	return func(i *ImportOptions) {
		i.caseInt = isCase
	}
}

type ImportOptions struct {
	dsn        string
	tables     []string
	schemaPath string
	caseInt    bool //default is true, int8,32,64 --> Int
}
