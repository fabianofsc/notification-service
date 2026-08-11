package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

func SendNotificationHandler(deps app.SendNotificationDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateNotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := req.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		recipient, err := domain.NewRecipient(req.Recipient)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid recipient: %v", err))
			return
		}

		input := app.SendNotificationInput{
			NotificationKey: req.NotificationKey,
			Channel:         domain.ChannelEmail,
			Recipient:       recipient,
			Subject:         req.Subject,
			Body:            req.Body,
			ReferenceID:     req.ReferenceID,
		}

		n, err := app.SendNotification(r.Context(), deps, input)
		if err != nil {
			if err == domain.ErrPayloadMismatch {
				writeJSONError(w, http.StatusConflict, "payload mismatch")
				return
			}
			slog.Error("failed to send notification", "error", err)
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
			return
		}

		status := http.StatusAccepted
		if n.CreatedAt != n.UpdatedAt {
			status = http.StatusOK
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(NewNotificationResponse(n))
	}
}

func GetNotificationHandler(deps app.GetNotificationDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		encodedID := r.PathValue("id")
		decodedID, err := decodeID(notificationKind, encodedID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "notification not found")
			return
		}

		n, err := app.GetNotification(r.Context(), deps, app.GetNotificationInput{ID: decodedID})
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "notification not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(NewNotificationResponse(n))
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}