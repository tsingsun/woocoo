// Code generated by woco, DO NOT EDIT.

package client

import (
	"encoding/json"
	"time"
)

// Category A category for a pet
type Category struct {
	ID   int64  `json:"id,omitempty" xml:"id"`
	Name string `binding:"omitempty,regex=oas_pattern_0" json:"name,omitempty" xml:"name"`
}

type NewPet struct {
	// Pet A pet for sale in the pet store
	*Pet `json:",inline"`
	// Owner A User who is purchasing from the pet store
	Owner     User      `binding:"required" json:"owner" xml:"User"`
	Timestamp time.Time `binding:"required" json:"timestamp" time_format:"2006-01-02T15:04:05Z07:00" xml:"timestamp"`
}

// Order An order for a pets from the pet store
type Order struct {
	Complete  bool      `json:"complete,omitempty" xml:"complete"`
	ID        int64     `json:"id,omitempty" xml:"id"`
	OrderDate time.Time `binding:"omitempty,ltfield=ShipDate" json:"orderDate,omitempty" time_format:"2006-01-02T15:04:05Z07:00" xml:"orderDate"`
	PetId     int64     `json:"petId,omitempty" xml:"petId"`
	Quantity  int32     `json:"quantity,omitempty" xml:"quantity"`
	ShipDate  time.Time `json:"shipDate,omitempty" time_format:"2006-01-02T15:04:05Z07:00" xml:"shipDate"`
	// Status Order Status
	Status string `json:"status,omitempty" xml:"status"`
}

// Pet A pet for sale in the pet store
type Pet struct {
	// Category A category for a pet
	Category  *Category `json:"category,omitempty" xml:"Category"`
	ID        int64     `json:"id,omitempty" xml:"id"`
	Name      string    `binding:"required" json:"name" xml:"name"`
	PhotoUrls []string  `binding:"required" json:"photoUrls" xml:"photoUrl"`
	// Status pet status in the store
	Status string `json:"status,omitempty" xml:"status"`
	Tags   []*Tag `json:"tags,omitempty" xml:"tag"`
}

// Tag A tag for a pet
type Tag struct {
	ID     int64    `json:"id,omitempty" xml:"id"`
	Labels LabelSet `json:"labels,omitempty" xml:"labels"`
	Name   string   `json:"name,omitempty" xml:"name"`
}

// User A User who is purchasing from the pet store
type User struct {
	Email     string `binding:"omitempty,email" json:"email,omitempty" xml:"email"`
	FirstName string `json:"firstName,omitempty" xml:"firstName"`
	ID        int64  `json:"id,omitempty" xml:"id"`
	LastName  string `json:"lastName,omitempty" xml:"lastName"`
	Password  string `json:"password,omitempty" xml:"password"`
	Phone     string `json:"phone,omitempty" xml:"phone"`
	// UserStatus User Status
	UserStatus int32  `json:"user_status,omitempty" xml:"user_status"`
	Username   string `json:"username,omitempty" xml:"username"`
}

// JsonObject A JSON object
type JsonObject json.RawMessage

type LabelSet map[string]string

type Pets []*Pet
