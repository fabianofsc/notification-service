package http

import (
	"net/http"

	"github.com/nexus-shopping/notification-service/internal/app"
)

type Dependencies struct {
	SendNotification app.SendNotificationDeps
	GetNotification  app.GetNotificationDeps
	BasicAuthUser    string
	BasicAuthPass    string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	v1 := http.NewServeMux()
	v1.HandleFunc("POST /v1/notifications", SendNotificationHandler(deps.SendNotification))
	v1.HandleFunc("GET /v1/notifications/{id}", GetNotificationHandler(deps.GetNotification))

	authenticated := BasicAuth(deps.BasicAuthUser, deps.BasicAuthPass)(v1)

	mux.Handle("/v1/", authenticated)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}