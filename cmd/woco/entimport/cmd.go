package entimport

import (
	"github.com/tsingsun/woocoo/cmd/woco/entimport/internal/driver"
	"github.com/urfave/cli/v2"
	"strings"
)

var EntImportCmd = &cli.Command{
	Name:  "entimport",
	Usage: "a tool for creating Ent schemas from existing SQL databases",
	Action: func(c *cli.Context) error {
		var tables []string
		// support ','
		if len(c.StringSlice("tables")) == 1 {
			for _, s := range strings.Split(c.StringSlice("tables")[0], ",") {
				tables = append(tables, s)
			}
		} else {
			tables = c.StringSlice("tables")
		}
		opts := driver.ImportOptions{
			Dialect:       c.String("dialect"),
			DSN:           c.String("dsn"),
			SchemaPath:    c.String("output"),
			Tables:        tables,
			GenGraphql:    c.Bool("gql"),
			GenProtoField: c.Bool("protobuf"),
			CaseInt:       c.Bool("UseInt"),
		}
		return generateSchema(opts)
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dialect",
			Value:   "mysql",
			Aliases: []string{"d"},
			Usage:   "database dialect",
		},
		&cli.StringFlag{
			Name:    "dsn",
			Usage:   "data source name (connection information)",
			EnvVars: []string{"IMPORT_DSN"},
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Value:   "./ent/schema",
			Usage:   "output path for ent schema",
		},
		&cli.StringSliceFlag{
			Name:  "tables",
			Usage: "comma-separated list of tables to inspect (all if empty)",
		},
		&cli.BoolFlag{
			Name:    "gql",
			Aliases: []string{"g"},
			Value:   false,
			Usage:   "generate graphql file",
		},
		&cli.BoolFlag{
			Name:    "protobuf",
			Aliases: []string{"p"},
			Value:   false,
			Usage:   "generate protobuf file",
		},
		&cli.BoolFlag{
			Name:    "UseInt",
			Aliases: []string{"i"},
			Value:   true,
			Usage:   "universal int type",
		},
	},
}
