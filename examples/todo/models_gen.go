// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package todo

import (
	"todo/ent/todo"
)

type TodoInput struct {
	Status   todo.Status `json:"status"`
	Priority *int        `json:"priority"`
	Text     string      `json:"text"`
	Parent   *int        `json:"parent"`
}
