package oasgen

import (
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/codegen"
	"github.com/urfave/cli/v2"
	"strings"
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
		&cli.StringSliceFlag{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   "external templates to execute",
		},
	},
	Action: func(c *cli.Context) (err error) {
		var opts []Option
		for _, tmpl := range c.StringSlice("template") {
			typ := "dir"
			if parts := strings.SplitN(tmpl, "=", 2); len(parts) > 1 {
				typ, tmpl = parts[0], parts[1]
			}
			switch typ {
			case "dir":
				opts = append(opts, TemplateDir(tmpl))
			case "file":
				opts = append(opts, TemplateFiles(tmpl))
			}
		}
		cfg := &codegen.Config{}
		cnfPath := c.String("config")
		err = LoadConfig(cfg, cnfPath)
		if err != nil {
			return err
		}
		return Generate(cfg.OpenAPISchema, cfg, opts...)
	},
}
