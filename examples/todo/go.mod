module todo

go 1.16

require (
	entgo.io/contrib v0.1.0
	entgo.io/ent v0.9.0
	github.com/99designs/gqlgen v0.14.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/tsingsun/woocoo v0.0.0-20210811025747-2005492e00f1
	github.com/vektah/gqlparser/v2 v2.2.0
	github.com/vmihailenco/msgpack/v5 v5.3.4
	go.uber.org/zap v1.19.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace github.com/tsingsun/woocoo => ../../
