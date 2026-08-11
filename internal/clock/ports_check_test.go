package clock_test

import (
	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/clock"
)

var (
	_ app.Clock       = clock.Real{}
	_ app.Clock       = &clock.Fake{}
	_ app.IDGenerator = clock.UUIDv7Generator{}
)