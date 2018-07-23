package middleware

import "middleware/notify"

func InitMiddleware() error {
	notify.BUS = notify.NewBus()

	return nil
}
