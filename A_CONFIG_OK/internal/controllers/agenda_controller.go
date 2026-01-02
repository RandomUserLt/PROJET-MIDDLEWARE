package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"fmt"

	"github.com/go-chi/chi/v5"
	"middleware/example/internal/models"
	"middleware/example/internal/services"
)


type AgendaController struct{ svc services.AgendaService }



func NewAgendaController(s services.AgendaService) *AgendaController { return &AgendaController{svc: s} }



func (c *AgendaController) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.list)
	r.Get("/{id}", c.get)
	r.Post("/", c.create)
	r.Put("/{id}", c.update)
	r.Delete("/{id}", c.delete)
	return r
}







// @Summary      Récupérer un agenda
// @Tags         agendas
// @Param        id   path      string  true  "ID agenda"
// @Produce      plain
// @Success      200  {string}  string
// @Failure      400  {object}  models.APIError
// @Failure      404  {object}  models.APIError
// @Router       /agendas/{id} [get]
func (c *AgendaController) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := c.svc.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if a == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w,
		"Agenda\n"+
			"Identifiant : %s\n"+
			"Nom         : %s\n",
		a.ID, a.Name,
	)
}

// create
// @Summary      Créer un agenda
// @Tags         agendas
// @Accept       json
// @Produce      json
// @Param        payload  body  models.Agenda  true  "Agenda"
// @Success      201  {object}  models.Agenda
// @Failure      400  {object}  models.APIError
// @Router       /agendas [post]
func (c *AgendaController) create(w http.ResponseWriter, r *http.Request) {
	var a models.Agenda
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := c.svc.Create(r.Context(), a); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w,
		"Agenda créé avec succès\n"+
			"Identifiant : %s\n"+
			"Nom         : %s\n",
		a.ID, a.Name,
	)
}




// @Summary      Mettre à jour un agenda
// @Tags         agendas
// @Accept       json
// @Produce      plain
// @Param        id      path      string        true  "ID agenda"
// @Param        agenda  body      models.Agenda true  "Agenda (le champ id du body est ignoré, l'URL fait foi)"
// @Success      200     {string}  string
// @Failure      400     {object}  models.APIError
// @Failure      404     {object}  models.APIError
// @Router       /agendas/{id} [put]
func (c *AgendaController) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var payload models.Agenda
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	payload.ID = id

	if err := c.svc.Update(r.Context(), payload); err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w,
		"Agenda mis à jour\n"+
			"Identifiant : %s\n"+
			"Nom         : %s\n",
		payload.ID, payload.Name,
	)
}




// @Summary      Supprimer un agenda
// @Tags         agendas
// @Produce      plain
// @Param        id   path      string  true  "ID agenda"
// @Success      200  {string}  string
// @Failure      400  {object}  models.APIError
// @Failure      404  {object}  models.APIError
// @Router       /agendas/{id} [delete]
func (c *AgendaController) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := c.svc.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "Agenda supprimé (id = %s)\n", id)
}


// @Summary      Lister les agendas
// @Tags         agendas
// @Produce      plain
// @Success      200  {string}  string
// @Failure      500  {object}  models.APIError
// @Router       /agendas [get]
func (c *AgendaController) list(w http.ResponseWriter, r *http.Request) {
	items, err := c.svc.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Liste des Agendas :\n"))
	for i, a := range items {
		line := fmt.Sprintf(
			"Agenda n°%d : Identifiant : %s | Nom : %s\n",
			i+1, a.ID, a.Name,
		)
		w.Write([]byte(line))
	}
}




func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	
}



func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, models.APIError{Message: msg})
}
