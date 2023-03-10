package helper

import (
	"math"
	"time"
)

const (
	StatusOK   = "OK"
	TimeOut    = 1 * time.Second
	MaxTimeOut = 1 * time.Hour
)

type FailureCountable interface {
	GetFailureCount() int64
	SetFailureCount(count int64)
}

type StatusValue interface {
	GetStatus() string
	SetStatus(val string)
}

type StatusValueFailureCountable interface {
	FailureCountable
	StatusValue
}

func SetFailureCount(fc FailureCountable) time.Duration {
	failures := fc.GetFailureCount()
	timeout := getTimeout(failures, TimeOut)
	failures++
	fc.SetFailureCount(failures)

	return timeout
}

func getTimeout(factor int64, baseDuration time.Duration) time.Duration {
	expTimeout := time.Duration(float64(baseDuration) * math.Pow(math.E, float64(factor+1)))

	if expTimeout > MaxTimeOut || expTimeout <= 0 {
		return MaxTimeOut
	}

	return expTimeout
}

func SetSuccessStatus(el StatusValueFailureCountable) {
	el.SetStatus(StatusOK)
	el.SetFailureCount(0)
}
