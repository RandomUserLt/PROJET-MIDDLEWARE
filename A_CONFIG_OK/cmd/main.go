package main
// @title           Config API
// @version         1.0
// @description     API de gestion des agendas et alertes
// @host            localhost:8080
// @BasePath        /
// @schemes         http

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"

	"middleware/example/internal/controllers"
	"middleware/example/internal/repositories"
	"middleware/example/internal/services"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "middleware/example/docs"

)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./config.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	agendaRepo := repositories.NewAgendaSQLite(db)
	if err := agendaRepo.Init(context.Background()); err != nil {
		log.Fatal(err)
	}
	agendaSvc := services.NewAgendaService(agendaRepo)
	agendaCtrl := controllers.NewAgendaController(agendaSvc)

	// --- Alerts wiring ---
	alertRepo := repositories.NewAlertSQLite(db)
	if err := alertRepo.Init(context.Background()); err != nil {
		log.Fatal(err)
	}
	alertSvc := services.NewAlertService(alertRepo, agendaRepo)
	alertCtrl := controllers.NewAlertController(alertSvc)

	r := chi.NewRouter()

    
   	 r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"), ))


	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Mount("/agendas", agendaCtrl.Routes())
	r.Mount("/alerts", alertCtrl.Routes())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("listening on :" + port)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
