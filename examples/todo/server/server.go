package main

import (
	"context"
	"entgo.io/contrib/entgql"
	"github.com/tsingsun/woocoo/pkg/log"
	"github.com/tsingsun/woocoo/web"
	"github.com/tsingsun/woocoo/web/handler/gql"
	"go.uber.org/zap"
	"todo"
	"todo/ent"
	"todo/ent/migrate"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tsingsun/woocoo/web/handler/gql"
	_ "todo/ent/runtime"
)

func main() {
	httpSvr := web.Default()
	//r := httpSvr.Router().Engine

	client, err := ent.Open(
		"sqlite3",
		"file:ent?mode=memory&cache=shared&_fk=1",
	)
	if err != nil {
		log.Fatal("opening ent client", zap.Error(err))
	}
	if err := client.Schema.Create(
		context.Background(),
		migrate.WithGlobalUniqueID(true),
	); err != nil {
		log.Fatal("running schema migration", zap.Error(err))
	}
	srv := gql.DefaultGraphqlServer(httpSvr, todo.NewSchema(client))
	//srv := handler.NewDefaultServer(todo.NewSchema(client))
	srv.Use(entgql.Transactioner{TxOpener: client})
	//if httpSvr.ServerConfig().Development {
	//	srv.Use(&debug.Tracer{})
	//}

	if err := httpSvr.Run(true); err != nil {
		log.Fatal(err)
	}
}
