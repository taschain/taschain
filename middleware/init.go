package middleware

import (
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/time"
)

func InitMiddleware() error {
	notify.BUS = notify.NewBus()
	time.InitTimeSync()
	return nil
}
