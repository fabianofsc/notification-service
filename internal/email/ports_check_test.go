package email_test

import (
	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/email"
)

var _ app.EmailProvider = &email.FakeProvider{}
