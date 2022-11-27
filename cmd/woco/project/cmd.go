package project

import (
	"github.com/urfave/cli/v2"
	"path/filepath"
)

var InitCmd = &cli.Command{
	Name:  "init",
	Usage: "a tool for generate woocoo web code from OpenAPI 3 specifications",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "package",
			Aliases:  []string{"p"},
			Usage:    "the package name of the generated code",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "target",
			Aliases:  []string{"t"},
			Usage:    "the target directory of the generated code",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:    "modules",
			Aliases: []string{"m"},
			Usage:   "the modules of the generated code",
		},
	},
	Action: func(c *cli.Context) (err error) {
		dir := c.String("target")
		// get full path by "."
		fd, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		cfg := &Config{
			Package: c.String("package"),
			Target:  fd,
			Modules: c.StringSlice("modules"),
		}

		return generateWeb(cfg)
	},
}
