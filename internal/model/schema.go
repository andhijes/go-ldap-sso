package model

import "time"

type Employee struct {
	ID    uint   `gorm:"primaryKey"`
	UID   string `gorm:"unique;not null"`
	Name  string
	Email string `gorm:"unique;not null"`
}

type Scope struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"unique;not null"`
	Description string
}

type EmployeeScope struct {
	EmployeeID uint
	ScopeID    uint
}

type TokenBlacklist struct {
	Token     string `gorm:"primaryKey"`
	ExpiresAt time.Time
}
