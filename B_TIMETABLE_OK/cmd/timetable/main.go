package main

// @title        API TIMETABLE 
// @version      1.0
// @description  API de lecture des évènements (iCal) de l'UCA.
// @schemes      http
// @BasePath     /

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"middleware/example/internal/controllers"
	"middleware/example/internal/services"
	_ "middleware/example/docs"
)

func main() {

	client := &http.Client{ Timeout: 15 * time.Second }

	svc := services.NewTimetableService(client)
	ctrl := controllers.NewTimetableController(svc)

	r := chi.NewRouter()

	r.Handle("/docs/*", http.StripPrefix("/docs", http.FileServer(http.Dir("./docs"))))
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/docs/swagger.json"),))

	r.Get("/events", ctrl.List)
	r.Get("/events/{id}", ctrl.GetByID)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	fmt.Println("Timetable API listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
