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
	// HTTP client with timeout to fetch the iCal
	client := &http.Client{ Timeout: 15 * time.Second }

	svc := services.NewTimetableService(client)
	ctrl := controllers.NewTimetableController(svc)

	r := chi.NewRouter()

	// Swagger (re-use the same /api swagger dir if you generate combined docs)
	r.Handle("/docs/*", http.StripPrefix("/docs", http.FileServer(http.Dir("./docs"))))
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/docs/swagger.json"),))

	// Routes
	r.Get("/events", ctrl.List)
	//new
	r.Get("/events/{id}", ctrl.GetByID)
	//id := chi.URLParam(r, "id")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	fmt.Println("Timetable API listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
