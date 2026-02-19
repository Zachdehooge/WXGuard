package main

import (
	"time"
)

func UTCTime() string {
	now := time.Now()
	return now.UTC().Format("15:04") + " UTC"
}
