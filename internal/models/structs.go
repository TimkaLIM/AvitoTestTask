package models

type Restaurant struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type MenuItem struct {
	ID           int    `json:"id"`
	RestaurantID int    `json:"restaurant_id"`
	Name         string `json:"name"`
	Price        int    `json:"price"`
}

type OrderItemRequest struct {
	MenuItemID int `json:"menu_item_id"`
	Quantity   int `json:"quantity"`
}

type CreateOrderRequest struct {
	UserID       int                `json:"user_id"`
	RestaurantID int                `json:"restaurant_id"`
	Items        []OrderItemRequest `json:"items"`
}

type PatchOrderStatusRequest struct {
	Status string `json:"status"`
}

type RestaurantOrders struct {
	OrderID    int    `json:"order_id"`
	UserID     int    `json:"user_id"`
	Status     string `json:"status"`
	TotalPrice int    `json:"total_price"`
}
