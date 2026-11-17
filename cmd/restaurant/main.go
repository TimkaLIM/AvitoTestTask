package main

import (
	"AvitoTestTask/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080" // для локального запуска
	}
	restaurantIDStr := os.Getenv("RESTAURANT_ID")
	if restaurantIDStr == "" {
		restaurantIDStr = "1" // По умолчанию работаем с первым рестораном
	}
	restaurantID, err := strconv.Atoi(restaurantIDStr)
	if err != nil {
		log.Println("Ошибка перевода ID ресторана в число: ", err)
		return
	}

	for {
		fmt.Println("Делаем запрос на поиск заказов...")
		url1 := fmt.Sprintf("%s/restaurant/orders?restaurant_id=%d&status=new", apiURL, restaurantID)
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
		resp.Body.Close()
		for _, value := range orders {

			url := fmt.Sprintf("%s/orders/%d/status", apiURL, value.OrderID)
			client := &http.Client{Timeout: 5 * time.Second}

			//готовка

			req, err := http.NewRequest("PATCH", url, strings.NewReader(`{"status": "cooking"}`))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp2, err := client.Do(req)
			if err != nil {
				log.Println("Ошибка отправки запроса: ", err)
				continue
			}
			resp2.Body.Close()
			fmt.Printf("Заказ с id = %d готовится...\n", value.OrderID)
			time.Sleep(3 * time.Second)

			//доставка

			req, err = http.NewRequest("PATCH", url, strings.NewReader(`{"status": "delivering"}`))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp2, err = client.Do(req)
			if err != nil {
				log.Println("Ошибка отправки запроса: ", err)
				continue
			}
			resp2.Body.Close()
			fmt.Printf("Заказ с id = %d отправлен в доствку...\n", value.OrderID)
			time.Sleep(3 * time.Second)

			//клиент получил заказ

			req, err = http.NewRequest("PATCH", url, strings.NewReader(`{"status": "done"}`))
			if err != nil {
				log.Println("Ошибка отправки PATCH запроса: ", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp2, err = client.Do(req)
			if err != nil {
				log.Println("Ошибка отправки запроса: ", err)
				continue
			}
			resp2.Body.Close()
			fmt.Printf("Заказ с id = %d доставлен!\n", value.OrderID)
			time.Sleep(3 * time.Second)
		}

		time.Sleep(5 * time.Second)

	}
}
