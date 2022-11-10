package oasgen

import (
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/codegen"
	"github.com/urfave/cli/v2"
)

var OasGenCmd = &cli.Command{
	Name:  "oasgen",
	Usage: "a tool for generate woocoo web code from OpenAPI 3 specifications",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Value:    "oasgen.yaml",
			Aliases:  []string{"c"},
			Usage:    "configuration file",
			Required: true,
		},
	},
	Action: func(c *cli.Context) (err error) {
		cfg := &codegen.Config{}
		cnfPath := c.String("config")
		err = LoadConfig(cfg, cnfPath)
		if err != nil {
			return err
		}
		return Generate(cfg.OpenAPISchema, cfg)
	},
}
