package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/adapter/inbound/http/dto"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/port"
)

// OfferingHandler serves the Courses page: offerings CRUD, bulk delete and the access list of an offering.
type OfferingHandler struct {
	list         port.ListOfferingsUseCase
	years        port.ListOfferingYearsUseCase
	create       port.CreateOfferingUseCase
	update       port.UpdateOfferingUseCase
	del          port.DeleteOfferingsUseCase
	listAccess   port.ListAccessUseCase
	setAccess    port.SetAccessUseCase
	removeAccess port.RemoveAccessUseCase
}

func NewOfferingHandler(
	list port.ListOfferingsUseCase,
	years port.ListOfferingYearsUseCase,
	create port.CreateOfferingUseCase,
	update port.UpdateOfferingUseCase,
	del port.DeleteOfferingsUseCase,
	listAccess port.ListAccessUseCase,
	setAccess port.SetAccessUseCase,
	removeAccess port.RemoveAccessUseCase,
) *OfferingHandler {
	return &OfferingHandler{
		list: list, years: years, create: create, update: update, del: del,
		listAccess: listAccess, setAccess: setAccess, removeAccess: removeAccess,
	}
}

// Register mounts the routes on the authenticated /api/v1 group. The static segments (years, bulk-delete)
// live next to the :id routes; Gin prefers the static ones.
func (h *OfferingHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/offerings")
	g.GET("", h.List)
	g.GET("/years", h.Years)
	g.POST("", h.Create)
	g.POST("/bulk-delete", h.BulkDelete)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.GET("/:id/access", h.ListAccess)
	g.PUT("/:id/access/:user_id", h.SetAccess)
	g.DELETE("/:id/access/:user_id", h.RemoveAccess)
}

func (h *OfferingHandler) List(c *gin.Context) {
	var in port.ListOfferingsInput
	rawYear := c.Query("academic_year")
	if rawYear == "" {
		badRequest(c, "academic_year is required")
		return
	}
	year, err := strconv.Atoi(rawYear)
	if err != nil {
		badRequest(c, "invalid academic_year")
		return
	}
	in.AcademicYear = year
	if in.DepartmentID, err = offeringQueryUUID(c, "department_id"); err != nil {
		badRequest(c, "invalid department_id")
		return
	}
	if in.CurriculumID, err = offeringQueryUUID(c, "curriculum_id"); err != nil {
		badRequest(c, "invalid curriculum_id")
		return
	}
	if raw := c.Query("semester"); raw != "" {
		sem, err := strconv.Atoi(raw)
		if err != nil {
			badRequest(c, "invalid semester")
			return
		}
		in.Semester = &sem
	}
	in.Search = c.Query("search")

	items, err := h.list.Execute(c.Request.Context(), middleware.ActorFrom(c), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.OfferingsFromDomain(items)))
}

func (h *OfferingHandler) Years(c *gin.Context) {
	years, current, err := h.years.Execute(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.YearsResponse{Items: years, Current: current})
}

func (h *OfferingHandler) Create(c *gin.Context) {
	var req dto.CreateOfferingRequest
	if !bindJSON(c, &req) {
		return
	}
	o, err := h.create.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.OfferingFromDomain(o))
}

func (h *OfferingHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateOfferingRequest
	if !bindJSON(c, &req) {
		return
	}
	o, err := h.update.Execute(c.Request.Context(), middleware.ActorFrom(c), req.ToInput(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OfferingFromDomain(o))
}

func (h *OfferingHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.del.Execute(c.Request.Context(), middleware.ActorFrom(c), []uuid.UUID{id}); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *OfferingHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !bindJSON(c, &req) {
		return
	}
	n, err := h.del.Execute(c.Request.Context(), middleware.ActorFrom(c), req.IDs)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.BulkDeleteResponse{Deleted: n})
}

func (h *OfferingHandler) ListAccess(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	items, err := h.listAccess.Execute(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Items(dto.AccessEntriesFromDomain(items)))
}

func (h *OfferingHandler) SetAccess(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := parseUUIDParam(c, "user_id")
	if !ok {
		return
	}
	var req dto.SetAccessRequest
	if !bindJSON(c, &req) {
		return
	}
	entry, err := h.setAccess.Execute(c.Request.Context(), middleware.ActorFrom(c), port.SetAccessInput{
		OfferingID: id, UserID: userID, CanUpload: req.CanUpload, CanDownload: req.CanDownload,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.AccessEntryFromDomain(*entry))
}

func (h *OfferingHandler) RemoveAccess(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	userID, ok := parseUUIDParam(c, "user_id")
	if !ok {
		return
	}
	if err := h.removeAccess.Execute(c.Request.Context(), middleware.ActorFrom(c), id, userID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// offeringQueryUUID reads an optional UUID query parameter (nil when absent).
func offeringQueryUUID(c *gin.Context, name string) (*uuid.UUID, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
