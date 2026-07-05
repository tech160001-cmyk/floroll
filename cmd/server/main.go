package main

import (
	"log"
	"net/http"
	"os"

	"floroll/internal/db"
	"floroll/internal/web"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "floroll.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("не удалось открыть базу данных: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("не удалось выполнить миграции: %v", err)
	}

	handler, err := web.NewHandler(database)
	if err != nil {
		log.Fatalf("не удалось инициализировать приложение: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("FloRoll запущен: http://localhost:%s", port)
	if err := http.ListenAndServe(addr, handler.Router()); err != nil {
		log.Fatal(err)
	}
}
