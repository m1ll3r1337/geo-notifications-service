package locations

import (
	"context"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type Repository interface {
	FindNearby(ctx context.Context, p Point, limit int) ([]NearbyIncident, error)
	RecordCheck(ctx context.Context, userID string, p Point, incidentIDs []int64) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) FindNearby(ctx context.Context, cmd CheckCommand) ([]NearbyIncident, error) {
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

func (s *Service) RecordCheck(ctx context.Context, userID string, p Point, incidentIDs []int64) error {
	const op = "location.service.record_check"

	if err := p.Validate(op); err != nil {
		return err
	}
	if err := s.repo.RecordCheck(ctx, userID, p, incidentIDs); err != nil {
		return errs.Wrap(op, err)
	}
	return nil
}
