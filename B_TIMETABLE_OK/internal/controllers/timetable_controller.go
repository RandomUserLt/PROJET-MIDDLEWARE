package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"middleware/example/internal/models"
	"middleware/example/internal/services"
	"fmt"
)

type TimetableController struct {
	svc services.TimetableService
}

func NewTimetableController(s services.TimetableService) *TimetableController {
	return &TimetableController{svc: s}
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "*", "")   
	s = strings.ReplaceAll(s, "\\n", "\n") 
	s = strings.TrimSpace(s)            
	return s
}

// List godoc
// @Summary      Lister les événements
// @Description  Récupère les événements iCal pour une liste d'agendas, avec filtre optionnel sur une plage de dates.
// @Tags         events
// @Produce      json
// @Param        agendaIds  query     string  true   "IDs d'agendas séparés par des virgules"  exemple("123,456")
// @Param        from      query     string  false  "Date début (YYYY-MM-DD)"                 exemple("2026-01-02")
// @Param        to        query     string  false  "Date fin (YYYY-MM-DD)"                   exemple("2026-01-10")
// @Success      200       {array}   models.Event
// @Failure      400       {object}  models.APIError
// @Failure      502       {object}  models.APIError
// @Router       /events [get]
func (c *TimetableController) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rawIDs := strings.TrimSpace(q.Get("agendaIds"))
	if rawIDs == "" {
		writeJSON(w, http.StatusBadRequest, models.APIError{Message: "agendaIds is required"})
		return
	}

	agendaIDs := []string{}
	for _, s := range strings.Split(rawIDs, ",") {
		s = strings.TrimSpace(s)
		if s != "" && s != "null" {
			agendaIDs = append(agendaIDs, s)
		}
	}

	fromPtr, toPtr, err := parseFromTo(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.APIError{Message: "invalid date format"})
		return
	}

	events, err := c.svc.FetchEvents(agendaIDs, fromPtr, toPtr)

	if err != nil {
		writeJSON(w, http.StatusBadGateway, models.APIError{Message: err.Error()})
		return
	}

	fmt.Fprintf(w, "=== Liste des événements ===\n")
	for i, ev := range events {
		startTime, _ := time.Parse(time.RFC3339, ev.Start)
		endTime, _ := time.Parse(time.RFC3339, ev.End)

		desc := strings.ReplaceAll(ev.Description, "\\n", "\n")
		desc = strings.ReplaceAll(desc, "*", "")
		desc = strings.TrimSpace(desc)

		fmt.Fprintf(w, "\nÉvénement #%d\n", i+1)
		fmt.Fprintf(w, "Titre      : %s\n", cleanText(ev.Title))
		fmt.Fprintf(w, "ID         : %s\n", cleanText(ev.ID))
		fmt.Fprintf(w, "Agendas    : %s\n", strings.Join(ev.AgendaIDs, ", "))
		fmt.Fprintf(w, "Début      : %s\n", startTime.Format("Monday 02 January 2006 15:04"))
		fmt.Fprintf(w, "Fin        : %s\n", endTime.Format("Monday 02 January 2006 15:04"))
		fmt.Fprintf(w, "Lieu       : %s\n", cleanText(ev.Location))
		fmt.Fprintf(w, "Description:\n%s\n", cleanText(desc))
	}
	
	
}


// GetByID godoc
// @Summary      Obtenir un événement par ID
// @Description  Cherche l'événement dans le flux iCal (agendaIds requis).
// @Tags         events
// @Produce      json
// @Param        id        path      string  true   "UID iCal de l'événement"
// @Param        agendaIds query     string  true   "IDs d'agendas séparés par des virgules"  example("123,456")
// @Success      200       {object}  models.Event
// @Failure      400       {object}  models.APIError
// @Failure      404       {object}  models.APIError
// @Failure      502       {object}  models.APIError
// @Router       /events/{id} [get]
func (c *TimetableController) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.APIError{Message: "invalid id"})
		return
	}

	agendaIdsParam := strings.TrimSpace(r.URL.Query().Get("agendaIds"))
	if agendaIdsParam == "" {
		writeJSON(w, http.StatusBadRequest, models.APIError{Message: "agendaIds required"})
		return
	}
	agendaIDs := strings.Split(agendaIdsParam, ",")

	
	events, err := c.svc.FetchEvents(agendaIDs, nil, nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, models.APIError{Message: "ical fetch failed: " + err.Error()})
		return
	}


	
	for _, ev := range events {
		 if strings.TrimSpace(ev.ID) == strings.TrimSpace(id){
		
			startTime, _ := time.Parse(time.RFC3339, ev.Start)
			endTime, _ := time.Parse(time.RFC3339, ev.End)

			desc := strings.ReplaceAll(ev.Description, "\\n", "\n")
			desc = strings.ReplaceAll(desc, "*", "")
			desc = strings.TrimSpace(desc)

			fmt.Fprintf(w, "=== Événement ===\n")
			fmt.Fprintf(w, "Titre      : %s\n", cleanText(ev.Title))
			fmt.Fprintf(w, "ID         : %s\n", cleanText(ev.ID))
			fmt.Fprintf(w, "Agendas    : %s\n", strings.Join(ev.AgendaIDs, ", "))
			fmt.Fprintf(w, "Début      : %s\n", startTime.Format("Monday 02 January 2006 15:04"))
			fmt.Fprintf(w, "Fin        : %s\n", endTime.Format("Monday 02 January 2006 15:04"))
			fmt.Fprintf(w, "Lieu       : %s\n", cleanText(ev.Location))
			fmt.Fprintf(w, "Description:\n%s\n", cleanText(desc))
			return
		}
	}

	fmt.Fprintln(w, "Événement non trouvé")
	
	
}





func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}


func parseFromTo(r *http.Request) (fromPtr, toPtr *time.Time, err error) {
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("from")); v != "" {
		t, e := time.Parse("2006-01-02", v)
		if e != nil {
			return nil, nil, e
		}
		ft := t.UTC()
		fromPtr = &ft
	}
	if v := strings.TrimSpace(q.Get("to")); v != "" {
		t, e := time.Parse("2006-01-02", v)
		if e != nil {
			return nil, nil, e
		}
		tt := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second).UTC()
		toPtr = &tt
	}
	return fromPtr, toPtr, nil
}

