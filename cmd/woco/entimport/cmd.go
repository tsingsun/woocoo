package entimport

import (
	"github.com/urfave/cli/v2"
)

var EntImportCmd = &cli.Command{
	Name:  "entimport",
	Usage: "a tool for creating Ent schemas from existing SQL databases",
	Action: func(c *cli.Context) error {
		dialect := c.String("dialect")
		dsn := c.String("dsn")
		output := c.String("output")
		tables := c.StringSlice("tables")
		return generateSchema(dialect, dsn, output, tables)
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
	},
}
