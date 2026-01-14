package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/locations"
	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type Locations struct {
	svc *locations.Service
}

func NewLocations(svc *locations.Service) *Locations {
	return &Locations{svc: svc}
}

type Location struct {
	Lat float64 `json:"lat" binding:"required"`
	Lon float64 `json:"lon" binding:"required"`
}

type locationCheckRequest struct {
	UserID   string   `json:"user_id" binding:"required"`
	Location Location `json:"location" binding:"required"`
	Limit    int      `json:"limit"`
}

type nearbyIncident struct {
	IncidentID     int64   `json:"incident_id"`
	DistanceMeters float64 `json:"distance_meters"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`

	Center Location `json:"center"`
	Radius int      `json:"radius"` // meters

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type locationCheckResponse struct {
	Count     int              `json:"count"`
	Incidents []nearbyIncident `json:"incidents"`
}

func (h *Locations) Check(ctx *gin.Context) {
	const op = "location.http.check"

	var req locationCheckRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_JSON", op, "invalid json", nil, err))
		return
	}

	cmd := locations.CheckCommand{
		UserID: req.UserID,
		Point:  locations.Point{Lat: req.Location.Lat, Lon: req.Location.Lon},
		Limit:  req.Limit,
	}

	items, err := h.svc.FindNearby(ctx.Request.Context(), cmd)
	if err != nil {
		ctx.Error(err)
		return
	}

	httpIncidents := make([]nearbyIncident, 0, len(items))
	for _, item := range items {
		httpIncidents = append(httpIncidents, nearbyIncident{
			IncidentID:     item.IncidentID,
			DistanceMeters: item.DistanceMeters,
			Title:          item.Title,
			Description:    item.Description,
			Center: Location{
				Lat: item.Center.Lat,
				Lon: item.Center.Lon,
			},
			Radius:    item.Radius,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, locationCheckResponse{
		Count:     len(items),
		Incidents: httpIncidents,
	})
}
