package utils

import (
	"time"
)

func Now() time.Time {
	return time.Now().UTC()
}

func Timestamp(interval ...time.Duration) int64 {
	now := Now()
	if len(interval) == 0 {
		return ToUnix(now)
	}

	return ToUnix(now.Add(-interval[0]))
}

func ToUnix(t time.Time) int64 {
	return t.UTC().UnixMicro()
}
