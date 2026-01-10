package services

import (
	"context"
	"errors"
	"regexp"

	"middleware/example/internal/models"
	"middleware/example/internal/repositories"
)

type AlertService interface {
	List(ctx context.Context, agendaID *string) ([]models.Alert, error)
	Get(ctx context.Context, id string) (*models.Alert, error)
	Create(ctx context.Context, a models.Alert) error
	Update(ctx context.Context, a models.Alert) error
	Delete(ctx context.Context, id string) error
}

type alertService struct {
	repo    repositories.AlertRepository
	agendas repositories.AgendaRepository 
}

func NewAlertService(r repositories.AlertRepository, agendaRepo repositories.AgendaRepository) AlertService {
	return &alertService{repo: r, agendas: agendaRepo}
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func validateAlert(a models.Alert) error {
	if a.ID == "" {
		return errors.New("id required")
	}
	if a.AgendaID == "" {
		return errors.New("agenda_id required")
	}
	if a.Target == "" {
		return errors.New("target required")
	}
	if !emailRe.MatchString(a.Target) {
		return errors.New("target must be a valid email")
	}
	if a.Condition == "" {
		a.Condition = "always"
	}
	switch a.Condition {
	case "always", "room_change", "time_change", "teacher_change":

	default:
		return errors.New("invalid condition")
	}
	return nil
}

func (s *alertService) ensureAgendaExists(ctx context.Context, id string) error {
	ag, err := s.agendas.Get(ctx, id)
	if err != nil {
		return err
	}
	if ag == nil {
		return errors.New("agenda not found")
	}
	return nil
}

func (s *alertService) List(ctx context.Context, agendaID *string) ([]models.Alert, error) {
	return s.repo.List(ctx, agendaID)
}

func (s *alertService) Get(ctx context.Context, id string) (*models.Alert, error) {
	if id == "" {
		return nil, errors.New("id required")
	}
	return s.repo.Get(ctx, id)
}

func (s *alertService) Create(ctx context.Context, a models.Alert) error {
	if err := validateAlert(a); err != nil {
		return err
	}
	if err := s.ensureAgendaExists(ctx, a.AgendaID); err != nil {
		return err
	}
	return s.repo.Create(ctx, a)
}

func (s *alertService) Update(ctx context.Context, a models.Alert) error {
	if err := validateAlert(a); err != nil {
		return err
	}
	if err := s.ensureAgendaExists(ctx, a.AgendaID); err != nil {
		return err
	}
	return s.repo.Update(ctx, a)
}

func (s *alertService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id required")
	}
	return s.repo.Delete(ctx, id)
}
