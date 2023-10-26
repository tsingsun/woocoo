package oasgen

import (
	"github.com/urfave/cli/v2"
	"strings"
)

var OasGenCmd = &cli.Command{
	Name:  "oasgen",
	Usage: "a tool for generate woocoo web code from OpenAPI 3 specifications",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "configuration file",
		},
		&cli.StringSliceFlag{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   "external templates to execute",
		},
		&cli.BoolFlag{
			Name:  "client",
			Value: false,
			Usage: "client side code generation",
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
		cfg := &Config{
			OpenAPISchema: "./openapi.yaml",
			GenClient:     c.Bool("client"),
		}
		cnfPath := c.String("config")
		err = LoadConfig(cfg, cnfPath)
		if err != nil {
			return err
		}
		return Generate(cfg.OpenAPISchema, cfg, opts...)
	},
}
