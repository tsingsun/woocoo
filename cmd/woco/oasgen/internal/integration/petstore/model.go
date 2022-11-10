// Code generated by woco, DO NOT EDIT.

package petstore

import "time"

type ApiResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Order struct {
	Complete bool      `json:"complete"`
	ID       int64     `json:"id"`
	PetId    int64     `json:"petId"`
	Quantity int32     `json:"quantity"`
	ShipDate time.Time `json:"shipDate"`
	Status   string    `json:"status"`
}

type Pet struct {
	Category  Category `json:"category"`
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	PhotoUrls []string `json:"photoUrls"`
	Status    string   `json:"status"`
	Tags      []Tag    `json:"tags"`
}

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type User struct {
	Email      string `json:"email"`
	FirstName  string `json:"firstName"`
	ID         int64  `json:"id"`
	LastName   string `json:"lastName"`
	Password   string `json:"password"`
	Phone      string `json:"phone"`
	UserStatus int32  `json:"userStatus"`
	Username   string `json:"username"`
}
