package domain

import "time"

type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusAccepted  OrderStatus = "accepted"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusPreparing OrderStatus = "preparing"
	OrderStatusReady     OrderStatus = "ready"
)

var orderTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusCreated:   {OrderStatusAccepted, OrderStatusRejected},
	OrderStatusAccepted:  {OrderStatusPreparing},
	OrderStatusPreparing: {OrderStatusReady},
	OrderStatusRejected:  nil,
	OrderStatusReady:     nil,
}

func (status OrderStatus) IsValid() bool {
	_, ok := orderTransitions[status]

	return ok
}

func (status OrderStatus) CanTransitionTo(target OrderStatus) bool {
	for _, allowed := range orderTransitions[status] {
		if allowed == target {
			return true
		}
	}

	return false
}

type OrderItem struct {
	MenuItemID       string
	ItemName         string
	UnitPriceKopecks int64
	Quantity         int
}

func (item OrderItem) TotalPriceKopecks() int64 {
	return item.UnitPriceKopecks * int64(item.Quantity)
}

type Order struct {
	ID                string
	RestaurantID      string
	UserID            string
	Status            OrderStatus
	TotalPriceKopecks int64
	DeliveryAddress   string
	Items             []OrderItem
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type OrderDraft struct {
	RestaurantID    string
	UserID          string
	IdempotencyKey  string
	DeliveryAddress string
	Items           []OrderItem
}

func (draft OrderDraft) TotalPriceKopecks() int64 {
	var total int64

	for _, item := range draft.Items {
		total += item.TotalPriceKopecks()
	}

	return total
}

func (draft OrderDraft) Matches(order Order) bool {
	if draft.RestaurantID != order.RestaurantID ||
		draft.DeliveryAddress != order.DeliveryAddress ||
		len(draft.Items) != len(order.Items) {
		return false
	}

	quantities := make(map[string]int, len(order.Items))

	for _, item := range order.Items {
		quantities[item.MenuItemID] = item.Quantity
	}

	for _, item := range draft.Items {
		quantity, ok := quantities[item.MenuItemID]
		if !ok || quantity != item.Quantity {
			return false
		}
	}

	return true
}
