package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

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

//HANDLER BLOCK END

//MIDDLEWARE BLOCK START

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Этот код выполняется до того, как попадет Handler(любой)
		start := time.Now()
		method := r.Method
		path := r.URL.Path
		w2 := &ResponseWriter{ResponseWriter: w, statusCode: 200}

		//Передаем управление дальше в хэндлер
		next.ServeHTTP(w2, r)

		status := w2.statusCode

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
