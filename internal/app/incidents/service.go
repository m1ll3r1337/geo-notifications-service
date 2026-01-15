package incidents

import (
	"context"
	"encoding/json"
	"time"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type IncidentsRepository interface {
	Create(ctx context.Context, in incidents.CreateIncident) (incidents.Incident, error)
	GetByID(ctx context.Context, id int64) (incidents.Incident, error)
	List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error)
	Update(ctx context.Context, id int64, in incidents.UpdateIncident) (incidents.Incident, error)
	Deactivate(ctx context.Context, id int64) error
	FindNearby(ctx context.Context, p incidents.Point, limit int) ([]incidents.NearbyIncident, error)
}

type Checker interface {
	RecordCheck(ctx context.Context, userID string, p incidents.Point, incidentIDs []int64) (int64, error)
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, eventType string, payloadJSON string, nextAttemptAt time.Time) error
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, checker Checker, outbox OutboxRepository) error) error
}

type Service struct {
	incRepo IncidentsRepository
	tx      TxRunner
}

func NewService(incRepo IncidentsRepository, tx TxRunner) *Service {
	return &Service{
		incRepo: incRepo,
		tx:      tx,
	}
}

func (s *Service) Create(ctx context.Context, cmd incidents.CreateIncident) (incidents.Incident, error) {
	const op = "incidents.service.create"

	if err := cmd.Validate(); err != nil {
		return incidents.Incident{}, errs.Wrap(op, err)
	}

	inc, err := s.incRepo.Create(ctx, cmd)
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

	inc, err := s.incRepo.GetByID(ctx, id)
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

	items, err := s.incRepo.List(ctx, f)
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

	inc, err := s.incRepo.Update(ctx, id, cmd)
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

	if err := s.incRepo.Deactivate(ctx, id); err != nil {
		return errs.Wrap(op, err)
	}

	return nil
}

type CheckResult struct {
	Incidents []incidents.NearbyIncident
	Count     int
}

func (s *Service) CheckAndRecord(ctx context.Context, cmd incidents.CheckCommand) (*CheckResult, error) {
	const op = "incidents.app.check_and_record"

	if err := cmd.Validate(); err != nil {
		return nil, errs.Wrap(op, err)
	}

	inc, err := s.incRepo.FindNearby(ctx, cmd.Point, cmd.Limit)
	if err != nil {
		return nil, errs.Wrap(op+".find_nearby", err)
	}

	incidentIDs := make([]int64, 0, len(inc))
	for _, it := range inc {
		incidentIDs = append(incidentIDs, it.IncidentID)
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context, checker Checker, outbox OutboxRepository) error {
		checkID, err := checker.RecordCheck(ctx, cmd.UserID, cmd.Point, incidentIDs)
		if err != nil {
			return errs.Wrap(op+".record_check", err)
		}

		if len(incidentIDs) == 0 {
			return nil
		}

		ev := incidents.CheckCompleted{
			CheckID:     checkID,
			UserID:      cmd.UserID,
			Point:       cmd.Point,
			IncidentIDs: incidentIDs,
			OccurredAt:  time.Now(),
		}

		b, err := json.Marshal(ev)
		if err != nil {
			return errs.Wrap(op+".marshal_event", err)
		}

		if err := outbox.Enqueue(ctx, "location_check", string(b), time.Now()); err != nil {
			return errs.Wrap(op+".enqueue_outbox", err)
		}

		return nil
	})
	if err != nil {
		return nil, errs.Wrap(op, err)
	}

	return &CheckResult{Incidents: inc, Count: len(inc)}, nil
}
