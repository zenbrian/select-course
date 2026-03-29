package course

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v5"
	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{
		service: service,
	}
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

func (h *handler) SelectCourse(w http.ResponseWriter, r *http.Request) {
	var body repo.CreateUserCourseParams
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	if body.UserID <= 0 || body.CourseID <= 0 {
		http.Error(w, "user_id and course_id must be > 0", http.StatusBadRequest)
		return
	}
	if course, err := h.service.SelectCourse(r.Context(), body.CourseID, body.UserID); err != nil {
		status, msg := mapCourseError(err)
		logCourseError("select", body.UserID, body.CourseID, err, status)
		http.Error(w, msg, status)
		return
	} else {
		if err := json.NewEncoder(w).Encode(course); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
func (h *handler) BackCourse(w http.ResponseWriter, r *http.Request) {
	var body repo.DeleteUserCourseParams
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	if body.UserID <= 0 || body.CourseID <= 0 {
		http.Error(w, "user_id and course_id must be > 0", http.StatusBadRequest)
		return
	}
	if course, err := h.service.BackCourse(r.Context(), body.CourseID, body.UserID); err != nil {
		status, msg := mapCourseError(err)
		logCourseError("back", body.UserID, body.CourseID, err, status)
		http.Error(w, msg, status)
		return
	} else {
		if err := json.NewEncoder(w).Encode(course); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func mapCourseError(err error) (int, string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return http.StatusNotFound, "user or course not found"
	case errors.Is(err, ErrCourseFull),
		errors.Is(err, ErrTimeConflict),
		errors.Is(err, ErrAlreadySelected),
		errors.Is(err, ErrNotSelected),
		errors.Is(err, ErrInvalidCourseWeek),
		errors.Is(err, ErrInvalidTimeSlot):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func logCourseError(action string, userID int64, courseID int64, err error, status int) {
	if status >= 500 {
		slog.Error("course operation failed", "action", action, "user_id", userID, "course_id", courseID, "status", status, "error", err)
		return
	}

	slog.Warn("course operation rejected", "action", action, "user_id", userID, "course_id", courseID, "status", status, "error", err)
}

// func (h *handler) CreateCourse(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) UpdateCourse(w http.ResponseWriter, r *http.Request) {}

// func (h *handler) DeleteCourse(w http.ResponseWriter, r *http.Request) {}
