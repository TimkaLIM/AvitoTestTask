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
