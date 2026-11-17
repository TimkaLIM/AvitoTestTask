package api

import (
	"AvitoTestTask/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

//structs BLOCK START

type Application struct {
	DB *sql.DB
}
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

//structs BLOCK END

//HANDLER BLOCK START

func (a *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	err := a.DB.Ping()
	if err != nil {
		log.Println("Ошибка связи с БД: ", err)
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(`{"status":"ok"}`))
}

func (a *Application) SelectRestaurants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	restaurants, err := a.DB.QueryContext(r.Context(), "SELECT id,title FROM restaurants")
	if err != nil {
		log.Println("Ошибка select ресторанов: ", err)
		w.WriteHeader(500)
		return
	}
	defer restaurants.Close()
	var restaurant_list []models.Restaurant

	for restaurants.Next() {
		restik := models.Restaurant{}
		if err = restaurants.Scan(&restik.ID, &restik.Title); err != nil {
			log.Println("Ошибка сканироваиня строки ресторана: ", err)
			break
		}
		restaurant_list = append(restaurant_list, restik)
	}
	if err = restaurants.Err(); err != nil {
		log.Println("Ошибка при чтении ресторанов: ", err)
		w.WriteHeader(500)
		return
	}
	err = json.NewEncoder(w).Encode(restaurant_list)
	if err != nil {
		log.Println("Ошибка энкодирования списка ресторанов: ", err)
		w.WriteHeader(500)
		return
	}
}

func (a *Application) SelectRestaurantMenu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Ошибка преобразования id из URL: ", err)
		w.WriteHeader(400)
		return
	}

	menu, err := a.DB.QueryContext(r.Context(), "SELECT id,restaurant_id,name,price FROM menu_items WHERE restaurant_id = $1", id)
	if err != nil {
		log.Println("Ошибка select элементов меню: ", err)
		w.WriteHeader(500)
		return
	}
	defer menu.Close()
	var menu_list []models.MenuItem
	for menu.Next() {

		menu_item := models.MenuItem{}

		if err = menu.Scan(&menu_item.ID, &menu_item.RestaurantID, &menu_item.Name, &menu_item.Price); err != nil {
			log.Println("Ошибка сканирования элементов меню: ", err)
			continue
		}
		menu_list = append(menu_list, menu_item)

	}
	if err = menu.Err(); err != nil {
		log.Println("Ошибка при чтении меню: ", err)
		w.WriteHeader(500)
		return
	}
	if err = json.NewEncoder(w).Encode(menu_list); err != nil {
		log.Println("Ошибка энкодирования элементов меню: ", err)
		w.WriteHeader(500)
		return
	}
}

func (a *Application) AddOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var order models.CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		log.Println("Ошибка декодирования заказа: ", err)
		w.WriteHeader(400)
		return
	}

	//Проверка, что заказ не пустой
	if len(order.Items) == 0 {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"empty order"}`))
		return
	}
	//считаем итоговую сумму заказа
	var MenuItemIDs []int
	for _, value := range order.Items {
		MenuItemIDs = append(MenuItemIDs, value.MenuItemID)
	}
	prices, err := a.DB.QueryContext(r.Context(), "SELECT id,price FROM menu_items WHERE id = ANY($1)", MenuItemIDs)
	if err != nil {
		log.Println("Ошибка SELECT prices: ", err)
		w.WriteHeader(500)
		return
	}
	defer prices.Close()

	priceMap := make(map[int]int)
	for prices.Next() {
		var id, price int
		err = prices.Scan(&id, &price)
		if err != nil {
			log.Println("Ошибка сканирования цены: ", err)
			continue
		}
		priceMap[id] = price
	}
	if prices.Err() != nil {
		log.Println("Ошибка при чтении цен: ", err)
		w.WriteHeader(500)
		return
	}
	var TotalPrice int
	for _, item := range order.Items {
		TotalPrice += priceMap[item.MenuItemID] * item.Quantity
	}

	//создаем транзакцию
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("Ошибка создания транзакции (BEGIN): ", err)
		w.WriteHeader(500)
		return
	}
	defer tx.Rollback()
	//добавляем элемент в заказ
	var orderID int
	err = tx.QueryRowContext(r.Context(), "INSERT INTO orders (restaurant_id,user_id,status,total_price) VALUES($1,$2,$3,$4) RETURNING id",
		order.RestaurantID, order.UserID, "new", TotalPrice).Scan(&orderID)
	if err != nil {
		log.Println("Ошибка создания заказа в orders: ", err)
		w.WriteHeader(500)
		return
	}
	//добавляем в таблице order_items все элементы
	n := len(order.Items)
	for i := 0; i < n; i++ {
		_, err = tx.ExecContext(r.Context(), "INSERT INTO order_items (order_id,menu_item_id,quantity) VALUES($1,$2,$3)",
			orderID, order.Items[i].MenuItemID, order.Items[i].Quantity)
		if err != nil {
			log.Println("Ошибка добавления элемента заказа: ", err)
			w.WriteHeader(500)
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Ошибка коммита транзакции: ", err)
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(`{"status":"order_is_created"}`))
}

func (a *Application) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	IDStr := r.PathValue("id")
	id, err := strconv.Atoi(IDStr)
	if err != nil {
		log.Println("Ошибка перевода IDStr в число: ", err)
		w.WriteHeader(400)
		return
	}
	var status string
	err = a.DB.QueryRowContext(r.Context(), "SELECT status FROM orders WHERE id = $1", id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Заказа нет — это 404!
			w.WriteHeader(http.StatusNotFound) // 404
			w.Write([]byte(`{"error":"order not found"}`))
			return
		}
		log.Println("Ошибка извлечения статуса заказа: ", err)
		w.WriteHeader(500)
		return
	}
	fmt.Fprintf(w, `{"status":%s}`, status)
}

func (a *Application) updateStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var NewStatus models.PatchOrderStatusRequest
	orderIDStr := r.PathValue("id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		log.Println("Ошибка перевода ID из строки в число: ", err)
		w.WriteHeader(400)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&NewStatus)
	if err != nil {
		log.Println("Ошибка декодирования нового статуса: ", err)
		w.WriteHeader(400)
		return
	}
	result, err := a.DB.ExecContext(r.Context(), "UPDATE orders SET status = $1 WHERE id = $2", NewStatus.Status, orderID)
	if err != nil {
		log.Println("Ошибка изменения статуса: ", err)
		w.WriteHeader(500)
		return
	}
	// Проверяем, обновилось ли что-то
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound) // 404
		w.Write([]byte(`{"error":"order not found"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}

func (a *Application) SelectRestaurantOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	restaurantIDStr := r.URL.Query().Get("restaurant_id")
	restaurantID, err := strconv.Atoi(restaurantIDStr)
	if err != nil {
		log.Println("Ошибка перевода ID из строки в число: ", err)
		w.WriteHeader(500)
		return
	}
	status := r.URL.Query().Get("status")
	query := "SELECT id, user_id, total_price FROM orders WHERE restaurant_id = $1"
	args := []any{restaurantID} // restaurantID должен быть int!

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	rows, err := a.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Println("Ошибка сканирования заказов №1: ", err)
		w.WriteHeader(500)
		return
	}
	var SelectRestaurants []models.RestaurantOrders
	for rows.Next() {
		var restaurant models.RestaurantOrders
		err = rows.Scan(&restaurant.OrderID, &restaurant.UserID, &restaurant.TotalPrice)
		if err != nil {
			log.Println("Ошибка сканирования заказов №1 ", err)
			w.WriteHeader(500)
			return
		}
		SelectRestaurants = append(SelectRestaurants, restaurant)
	}
	if rows.Err() != nil {
		log.Println("Ошибка при чтении заказов ресторана: ", rows.Err())
		w.WriteHeader(500)
		return
	}
	err = json.NewEncoder(w).Encode(SelectRestaurants)
	if err != nil {
		log.Println("Ошибка энкодинга ресторанов: ", err)
		w.WriteHeader(500)
		return
	}

}

//HANDLER BLOCK END

//MIDDLEWARE BLOCK START

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Этот код выполняется до того, как попадет Handler(любой)
		start := time.Now()
		method := r.Method
		path := r.URL.Path
		w2 := &ResponseWriter{ResponseWriter: w, StatusCode: 200}

		//Передаем управление дальше в хэндлер
		next.ServeHTTP(w2, r)

		status := w2.StatusCode

		//Этот код выполнится после того как хэндлер завершится
		duration := time.Since(start)
		fmt.Printf("[LOG] %s %s %d| Время: %v\n", method, path, status, duration)
	})
}

func registerRoute(mux *http.ServeMux, methodAndPath string,
	handler func(w http.ResponseWriter, r *http.Request)) {

	logger := loggingMiddleware(http.HandlerFunc(handler))
	mux.Handle(methodAndPath, logger)
}

func ServerStart(db *sql.DB) {
	mux := http.NewServeMux()
	app := &Application{
		DB: db,
	}
	registerRoute(mux, "GET /health", app.healthHandler)

	registerRoute(mux, "GET /restaurants", app.SelectRestaurants)
	registerRoute(mux, "GET /restaurants/{id}/menu", app.SelectRestaurantMenu)
	registerRoute(mux, "POST /orders", app.AddOrder)
	registerRoute(mux, "PATCH /orders/{id}/status", app.updateStatus)
	registerRoute(mux, "GET /orders/{id}", app.GetOrderStatus)
	registerRoute(mux, "GET /restaurant/orders", app.SelectRestaurantOrders)

	//ОТКРЫТИЕ СЕРВЕРА
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,   //максимум 5 сек на чтение запроса
		WriteTimeout: 10 * time.Second,  //Максимум 10 сек на отправку ответа
		IdleTimeout:  120 * time.Second, //Время жизни keep-alive соединения
	}

	//в горутине, для shutdown
	go func() {
		fmt.Println("Сервер запущен на :8080")
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed { //Проверка сразу 2 ошибок, просто ошибка, если правда что-то не так
			log.Fatalf("Ошибка сервера: %v", err) //Чтобы эта ошибка не выводилась при намеренном закрытии сервера череза ctrl C
		}
	}()
	//ждем сигнал Ctrl+C
	quit := make(chan os.Signal, 1)   //канал для сигналов ОС
	signal.Notify(quit, os.Interrupt) //если придет сигнал ctrl+C(interrupt) положи его в канал quit
	<-quit                            //программа замирает здесь, пока ты не нажмешь Ctrl+C

	log.Println("Сервер выключается")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil { //принудительное заверешние сервера, запрещает новые сообщения
		log.Fatal("Принудительное завершение: ", err) //+ ждет контекст 5 секунд
	}
	log.Println("Сервер успешно остановлен")
}
