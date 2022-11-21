// Code generated by woco, DO NOT EDIT.

package petstore

import "time"

type ApiResponse struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

type Category struct {
	ID   int64  `json:"id,omitempty" xml:"id"`
	Name string `binding:"regex=oas_pattern_0,omitempty" json:"name,omitempty" xml:"name"`
}

type Order struct {
	Complete bool      `json:"complete,omitempty" xml:"complete"`
	ID       int64     `json:"id,omitempty" xml:"id"`
	PetId    int64     `json:"petId,omitempty" xml:"petId"`
	Quantity int32     `json:"quantity,omitempty" xml:"quantity"`
	ShipDate time.Time `binding:"datetime=2006-01-02T15:04:05Z07:00,omitempty" json:"shipDate,omitempty" xml:"shipDate"`
	Status   string    `json:"status,omitempty" xml:"status"`
}

type Pet struct {
	Category  *Category `json:"category,omitempty" xml:"Category"`
	ID        int64     `json:"id,omitempty" xml:"id"`
	Name      string    `binding:"required" json:"name" xml:"name"`
	PhotoUrls []string  `binding:"required" json:"photoUrls" xml:"photoUrl"`
	Status    string    `json:"status,omitempty" xml:"status"`
	Tags      []Tag     `json:"tags,omitempty" xml:"tag"`
}

type Tag struct {
	ID   int64  `json:"id,omitempty" xml:"id"`
	Name string `json:"name,omitempty" xml:"name"`
}

type User struct {
	Email      string `binding:"email,omitempty" json:"email,omitempty" xml:"email"`
	FirstName  string `json:"firstName,omitempty" xml:"firstName"`
	ID         int64  `json:"id,omitempty" xml:"id"`
	LastName   string `json:"lastName,omitempty" xml:"lastName"`
	Password   string `json:"password,omitempty" xml:"password"`
	Phone      string `json:"phone,omitempty" xml:"phone"`
	UserStatus int32  `json:"userStatus,omitempty" xml:"userStatus"`
	Username   string `json:"username,omitempty" xml:"username"`
}
