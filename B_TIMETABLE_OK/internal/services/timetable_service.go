package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"middleware/example/internal/models"
)

// Interface exportée
type TimetableService interface {
	FetchEvents(agendaIDs []string, from, to *time.Time) ([]models.Event, error)
}

// Type concret non-exporté
type timetableService struct {
	client     *http.Client
	IcalBase   string
	WeeksParam string
}

// Constructeur
func NewTimetableService(client *http.Client) TimetableService {
	return &timetableService{
		client:     client,
		IcalBase:   "https://edt.uca.fr/jsp/custom/modules/plannings/anonymous_cal.jsp?projectId=3&calType=ical&displayConfigId=128",
		WeeksParam: "8",
	}
}

// Helper pour construire l'URL iCal
func (s *timetableService) buildURL(agendaIDs []string) (string, error) {
	if len(agendaIDs) == 0 {
		return "", errors.New("agendaIds required")
	}
	joined := strings.Join(agendaIDs, ",")
	return fmt.Sprintf("%s&nbWeeks=%s&resources=%s", s.IcalBase, s.WeeksParam, joined), nil
}

// Méthode principale
func (s *timetableService) FetchEvents(agendaIDs []string, from, to *time.Time) ([]models.Event, error) {
	var allEvents []models.Event

	url, err := s.buildURL(agendaIDs)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("ical fetch %d: %s", resp.StatusCode, string(b))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse calendar correctement
	cal, err := ics.ParseCalendar(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for _, e := range cal.Events() {
		layout := "20060102T150405Z" // format iCal UTC

		startTime, _ := time.Parse(layout, e.GetProperty("DTSTART").Value)
		endTime, _ := time.Parse(layout, e.GetProperty("DTEND").Value)
		lastMod, _ := time.Parse(layout, e.GetProperty("LAST-MODIFIED").Value)

		if from != nil && startTime.Before(*from) {
			continue
		}
		if to != nil && endTime.After(*to) {
			continue
		}

		event := models.Event{
			ID:          e.GetProperty("UID").Value,
			AgendaIDs:   agendaIDs,
			Title:       e.GetProperty("SUMMARY").Value,
			Description: e.GetProperty("DESCRIPTION").Value,
			Start:       startTime.Format(time.RFC3339),
			End:         endTime.Format(time.RFC3339),
			Location:    e.GetProperty("LOCATION").Value,
			LastUpdate:  lastMod.Format(time.RFC3339),
		}
		allEvents = append(allEvents, event)
	}

	if allEvents == nil {
		allEvents = []models.Event{}
	}

	return allEvents, nil
}

