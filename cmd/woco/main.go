package main

import (
	"github.com/tsingsun/woocoo/cmd/woco/entimport"
	"github.com/urfave/cli/v2"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const WoCo_Version = "0.0.1"

var commands = []*cli.Command{
	entimport.EntImportCmd,
}

func main() {
	app := cli.NewApp()
	app.Name = "woco"
	app.Usage = "a cli command for woocoo application"
	app.Version = WoCo_Version
	app.Commands = commands
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
}