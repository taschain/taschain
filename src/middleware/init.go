package middleware

import (
	"middleware/notify"
	"middleware/time"
)

func InitMiddleware() error {
	notify.BUS = notify.NewBus()
	time.InitTimeSync()
	return nil
}
