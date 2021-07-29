package handler

import (
	"github.com/tsingsun/woocoo/test/testdata"
	"testing"
)

var (
	cnf = testdata.Config
)

func TestAuthHandler(t *testing.T) {
	AuthHandler(cnf)

}
