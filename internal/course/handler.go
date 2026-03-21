package course

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{
		service: service,
	}
}

func (h *handler) CreateCourse(w http.ResponseWriter, r *http.Request) {
}

func (h *handler) GetCourse(w http.ResponseWriter, r *http.Request) {
	//json payload validation
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest) // 400
		return
	}
	course, err := h.service.GetCourseByID(r.Context(), id)
	if err != nil {
		http.Error(w, "course not found", http.StatusNotFound) //404
		return
	}
	if err := json.NewEncoder(w).Encode(course); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError) // 500
		return
	}
}

func (h *handler) UpdateCourse(w http.ResponseWriter, r *http.Request) {}

func (h *handler) DeleteCourse(w http.ResponseWriter, r *http.Request) {}
