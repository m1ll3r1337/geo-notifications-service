package incidents

import "time"

type CheckCompleted struct {
	CheckID     int64
	UserID      string
	Point       Point
	IncidentIDs []int64
	OccurredAt  time.Time
}
