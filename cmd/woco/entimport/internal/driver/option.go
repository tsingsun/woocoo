package driver

// ImportOptions holds the options for the import command.
type ImportOptions struct {
	Dialect       string
	DSN           string
	Tables        []string
	SchemaPath    string
	CaseInt       bool //default is true, int8,32,64 --> Int
	GenProtoField bool //default is false, generate proto field order
	GenGraphql    bool //default is false, generate graphql
}
