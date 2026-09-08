package domain

import "time"

type MenuItem struct {
	ID           string
	RestaurantID string
	ExternalID   string
	Name         string
	Description  string
	PriceKopecks int64
	IsAvailable  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type MenuSearchItem struct {
	MenuItem

	RestaurantName string
}

type MenuItemInput struct {
	ExternalID   string
	Name         string
	Description  string
	PriceKopecks int64
	IsAvailable  bool
}
