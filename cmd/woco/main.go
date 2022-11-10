package main

import (
	"github.com/tsingsun/woocoo/cmd/woco/entimport"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen"
	"github.com/urfave/cli/v2"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const WocoVersion = "0.0.1"

var commands = []*cli.Command{
	entimport.EntImportCmd,
	oasgen.OasGenCmd,
}

func main() {
	app := cli.NewApp()
	app.Name = "woco"
	app.Usage = "a cli command for woocoo application"
	app.Version = WocoVersion
	app.Commands = commands
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err.Error())
	}
}
