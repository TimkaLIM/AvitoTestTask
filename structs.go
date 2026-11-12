package main

import (
	"database/sql"
	"net/http"
)

type Application struct {
	DB *sql.DB
}

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}
