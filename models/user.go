package models

import (
	"github.com/golang-jwt/jwt/v4"
)

// User represents a User schema
type User struct {
	UserBase
	Email    string `json:"email" gorm:"unique"`
	Username string `json:"username" gorm:"unique"`
	Password string `json:"password"`
}

// UserErrors represent the error format for user routes
type UserErrors struct {
	Err      bool   `json:"error"`
	Msg 	string `json:"msg"`
}

// Claims represent the structure of the JWT token
type Claims struct {
	jwt.StandardClaims
	ID uint `gorm:"primaryKey"`
}
