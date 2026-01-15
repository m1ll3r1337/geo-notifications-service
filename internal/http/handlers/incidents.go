package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	incidentsapp "github.com/m1ll3r1337/geo-notifications-service/internal/app/incidents"
	incidentsdom "github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type Incidents struct {
	svc *incidentsapp.Service
}

func NewIncidents(svc *incidentsapp.Service) *Incidents {
	return &Incidents{svc: svc}
}

type point struct {
	Lat float64 `json:"lat" binding:"required"`
	Lon float64 `json:"lon" binding:"required"`
}

type incidentResponse struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Center      point     `json:"center"`
	Radius      int       `json:"radius"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toIncidentResponse(in incidentsdom.Incident) incidentResponse {
	return incidentResponse{
		ID:          in.ID,
		Title:       in.Title,
		Description: in.Description,
		Center:      point{Lat: in.Center.Lat, Lon: in.Center.Lon},
		Radius:      in.Radius,
		Active:      in.Active,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
	}
}

type createIncidentRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Center      point  `json:"center" binding:"required"`
	Radius      int    `json:"radius" binding:"required"`
}

func (h *Incidents) Create(ctx *gin.Context) {
	const op = "incidents.http.create"

	var req createIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_JSON", op, "invalid json", nil, err))
		return
	}

	inc, err := h.svc.Create(ctx, incidentsdom.CreateIncident{
		Title:       req.Title,
		Description: req.Description,
		Center: incidentsdom.Point{
			Lat: req.Center.Lat,
			Lon: req.Center.Lon,
		},
		Radius: req.Radius,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, toIncidentResponse(inc))
}

func (h *Incidents) GetByID(ctx *gin.Context) {
	const op = "incidents.http.get_by_id"

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, err))
		return
	}

	inc, err := h.svc.GetByID(ctx, id)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, toIncidentResponse(inc))
}

func (h *Incidents) List(ctx *gin.Context) {
	const op = "incidents.http.list"

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	activeOnly, _ := strconv.ParseBool(ctx.DefaultQuery("active_only", "true"))

	items, err := h.svc.List(ctx, incidentsdom.ListFilter{
		Limit:      limit,
		Offset:     offset,
		ActiveOnly: activeOnly,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	out := make([]incidentResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toIncidentResponse(it))
	}
	ctx.JSON(http.StatusOK, out)
}

type updateIncidentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Center      *point  `json:"center"`
	Radius      *int    `json:"radius"`
}

func (h *Incidents) Update(ctx *gin.Context) {
	const op = "incidents.http.update"

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, err))
		return
	}

	var req updateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_JSON", op, "invalid json", nil, err))
		return
	}

	var center *incidentsdom.Point
	if req.Center != nil {
		center = &incidentsdom.Point{Lat: req.Center.Lat, Lon: req.Center.Lon}
	}

	inc, err := h.svc.Update(ctx.Request.Context(), id, incidentsdom.UpdateIncident{
		Title:       req.Title,
		Description: req.Description,
		Center:      center,
		Radius:      req.Radius,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, toIncidentResponse(inc))
}

func (h *Incidents) Deactivate(ctx *gin.Context) {
	const op = "incidents.http.deactivate"

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_ID", op, "invalid id", map[string]string{"id": "must be > 0"}, err))
		return
	}

	if err := h.svc.Deactivate(ctx.Request.Context(), id); err != nil {
		ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

type locationCheckRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Location point  `json:"location" binding:"required"`
	Limit    int    `json:"limit"`
}

type nearbyIncident struct {
	IncidentID     int64   `json:"incident_id"`
	DistanceMeters float64 `json:"distance_meters"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`

	Center point `json:"center"`
	Radius int   `json:"radius"` // meters

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type locationCheckResponse struct {
	Count     int              `json:"count"`
	Incidents []nearbyIncident `json:"incidents"`
}

func (h *Incidents) Check(ctx *gin.Context) {
	const op = "location.http.check"

	var req locationCheckRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errs.E(errs.KindInvalid, "INVALID_JSON", op, "invalid json", nil, err))
		return
	}

	cmd := incidentsdom.CheckCommand{
		UserID: req.UserID,
		Point:  incidentsdom.Point{Lat: req.Location.Lat, Lon: req.Location.Lon},
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
			Center: point{
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
