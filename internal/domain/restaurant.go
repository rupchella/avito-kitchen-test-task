package domain

import "time"

type RestaurantStatus string

const (
	RestaurantStatusActive   RestaurantStatus = "active"
	RestaurantStatusInactive RestaurantStatus = "inactive"
)

type Restaurant struct {
	ID        string
	Name      string
	Status    RestaurantStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
