package postgres_test

import (
	"github.com/nexus-shopping/notification-service/internal/app"
	pg "github.com/nexus-shopping/notification-service/internal/postgres"
)

var (
	_ app.NotificationRepository = &pg.Repository{}
	_ app.DeliveryRepository     = &pg.Repository{}
)
