package incidents

import (
	"context"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type Repository interface {
	Create(ctx context.Context, in incidents.CreateIncident) (incidents.Incident, error)
	GetByID(ctx context.Context, id int64) (incidents.Incident, error)
	List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error)
	Update(ctx context.Context, id int64, in incidents.UpdateIncident) (incidents.Incident, error)
	Deactivate(ctx context.Context, id int64) error
	FindNearby(ctx context.Context, p incidents.Point, limit int) ([]incidents.NearbyIncident, error)
	RecordCheck(ctx context.Context, userID string, p incidents.Point, incidentIDs []int64) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, cmd incidents.CreateIncident) (incidents.Incident, error) {
	const op = "incidents.service.create"

	if err := cmd.Validate(); err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	inc, err := s.repo.Create(ctx, cmd)
	if err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	return inc, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (incidents.Incident, error) {
	const op = "incidents.service.get_by_id"

	if id <= 0 {
		return incidents.Incident{}, errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, nil)
	}

	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	return inc, nil
}

func (s *Service) List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error) {
	const op = "incidents.service.list"

	if f.Limit < 0 {
		f.Limit = 0
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	items, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}

	return items, nil
}

func (s *Service) Update(ctx context.Context, id int64, cmd incidents.UpdateIncident) (incidents.Incident, error) {
	const op = "incidents.service.update"

	if id <= 0 {
		return incidents.Incident{}, errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, nil)
	}
	if err := cmd.Validate(); err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	inc, err := s.repo.Update(ctx, id, cmd)
	if err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	return inc, nil
}

func (s *Service) Deactivate(ctx context.Context, id int64) error {
	const op = "incidents.service.deactivate"

	if id <= 0 {
		return errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, nil)
	}

	if err := s.repo.Deactivate(ctx, id); err != nil {
		return errs.Wrap(op, err)
	}

	return nil
}

func (s *Service) FindNearby(ctx context.Context, cmd incidents.CheckCommand) ([]incidents.NearbyIncident, error) {
	const op = "location.service.find_nearby"

	if err := cmd.Validate(); err != nil {
		return nil, errs.Wrap(op, err)
	}

	items, err := s.repo.FindNearby(ctx, cmd.Point, cmd.Limit)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	return items, nil
}

func (s *Service) RecordCheck(ctx context.Context, userID string, p incidents.Point, incidentIDs []int64) error {
	const op = "location.service.record_check"

	if err := p.Validate(op); err != nil {
		return err
	}
	if err := s.repo.RecordCheck(ctx, userID, p, incidentIDs); err != nil {
		return errs.Wrap(op, err)
	}
	return nil
}
