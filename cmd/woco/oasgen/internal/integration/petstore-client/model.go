// Code generated by woco, DO NOT EDIT.

package client

import (
	"encoding/json"
	"fmt"
	"time"
)

// Category A category for a pet
type Category struct {
	ID   int64  `json:"id,omitempty" xml:"id"`
	Name string `binding:"omitempty,regex=oas_pattern_0" json:"name,omitempty" xml:"name"`
}

// EnumObject defines the type for the EnumObject.EnumObject enum field.
type EnumObject string

// EnumObject values.
const (
	EnumObjectField1 EnumObject = "field1"
	EnumObjectField2 EnumObject = "field2"
	EnumObjectField3 EnumObject = "field3"
)

func (eo EnumObject) String() string {
	return string(eo)
}

// EnumObjectValidator is a validator for the EnumObject field enum values.
func EnumObjectValidator(eo EnumObject) error {
	switch eo {
	case EnumObjectField1, EnumObjectField2, EnumObjectField3:
		return nil
	default:
		return fmt.Errorf("EnumObject does not allow the value '%s'", eo)
	}
}

type LabelSet map[string]string

type NewPet struct {
	// Pet A pet for sale in the pet store
	*Pet `json:",inline"`
	// Owner A User who is purchasing from the pet store
	Owner     User      `binding:"required" json:"owner"`
	Timestamp time.Time `binding:"required" json:"timestamp" time_format:"2006-01-02T15:04:05Z07:00"`
}

// Order An order for a pets from the pet store
type Order struct {
	Complete  bool       `json:"complete,omitempty" xml:"complete"`
	ID        int64      `json:"id,omitempty" xml:"id"`
	OrderDate *time.Time `binding:"omitempty,ltfield=ShipDate" json:"orderDate,omitempty" time_format:"2006-01-02T15:04:05Z07:00" xml:"orderDate"`
	PetId     int64      `json:"petId,omitempty" xml:"petId"`
	Quantity  int32      `json:"quantity,omitempty" xml:"quantity"`
	ShipDate  *time.Time `json:"shipDate,omitempty" time_format:"2006-01-02T15:04:05Z07:00" xml:"shipDate"`
	// Status Order Status
	Status OrderStatus `binding:"omitempty,oneof=placed approved delivered" json:"status,omitempty" xml:"status"`
}

// Pet A pet for sale in the pet store
type Pet struct {
	// Category A category for a pet
	Category  *Category `json:"category,omitempty" xml:"Category"`
	ID        int64     `json:"id,omitempty" xml:"id"`
	Name      string    `binding:"required" json:"name" xml:"name"`
	PhotoUrls []string  `binding:"required" json:"photoUrls" xml:"photoUrl"`
	// Status pet status in the store
	Status PetStatus `binding:"omitempty,oneof=available pending sold" json:"status,omitempty" xml:"status"`
	Tags   []*Tag    `json:"tags,omitempty" xml:"tag"`
}

// Tag A tag for a pet
type Tag struct {
	ID     int64
	Labels LabelSet
	Lang   *Lang
	Name   string
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

type Lang struct {
	Code string
	Name string
}

type NewPets []*NewPet

type Pets []*Pet

// PetStatus defines the type for the status.status enum field.
type PetStatus string

// PetStatus values.
const (
	PetStatusAvailable PetStatus = "available"
	PetStatusPending   PetStatus = "pending"
	PetStatusSold      PetStatus = "sold"
)

func (s PetStatus) String() string {
	return string(s)
}

// PetStatusValidator is a validator for the PetStatus field enum values.
func PetStatusValidator(s PetStatus) error {
	switch s {
	case PetStatusAvailable, PetStatusPending, PetStatusSold:
		return nil
	default:
		return fmt.Errorf("PetStatus does not allow the value '%s'", s)
	}
}

// OrderStatus defines the type for the status.status enum field.
type OrderStatus string

// OrderStatus values.
const (
	OrderStatusPlaced    OrderStatus = "placed"
	OrderStatusApproved  OrderStatus = "approved"
	OrderStatusDelivered OrderStatus = "delivered"
)

func (s OrderStatus) String() string {
	return string(s)
}

// OrderStatusValidator is a validator for the OrderStatus field enum values.
func OrderStatusValidator(s OrderStatus) error {
	switch s {
	case OrderStatusPlaced, OrderStatusApproved, OrderStatusDelivered:
		return nil
	default:
		return fmt.Errorf("OrderStatus does not allow the value '%s'", s)
	}
}
