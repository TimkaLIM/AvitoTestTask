package main

import (
	"AvitoTestTask/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080" // для локального запуска
	}
	for {
		fmt.Println("Делаем запрос на поиск заказов...")
		url1 := fmt.Sprintf("%s/restaurant/orders?restaurant_id=1&status=new", apiURL)
		resp, err := http.Get(url1)
		if err != nil {
			log.Println("Ошибка запроса к заказам: ", err)
			continue
		}
		var orders []models.RestaurantOrders
		err = json.NewDecoder(resp.Body).Decode(&orders)
		if err != nil {
			log.Println("Ошибка декодирования: ", err)
			continue
		}
		for _, value := range orders {
			if value.Status == "cooking" || value.Status == "delivering" || value.Status == "pass" {
				continue
			}
			url := fmt.Sprintf("%s/orders/%d/status", apiURL, value.OrderID)
			//готовка
			_, err = http.NewRequest("PATCH", url, strings.NewReader("cooking"))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			fmt.Printf("Заказ с id = %d готовится...", value.OrderID)
			time.Sleep(3 * time.Second)
			//доставка
			_, err = http.NewRequest("PATCH", url, strings.NewReader("delivering"))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			fmt.Printf("Заказ с id = %d отправлен на доставку...", value.OrderID)
			time.Sleep(3 * time.Second)
			//клиент получил заказ
			_, err = http.NewRequest("PATCH", url, strings.NewReader("pass"))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			fmt.Printf("Заказ с id = %d доставлен!", value.OrderID)
			time.Sleep(3 * time.Second)
		}

		time.Sleep(5 * time.Second)

	}
}
