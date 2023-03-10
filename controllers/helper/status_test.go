package helper

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
)

func TestSetFailureCount(t *testing.T) {
	t.Parallel()

	type args struct {
		fc FailureCountable
	}

	var instance = gerritApi.GerritGroupMember{
		Status: gerritApi.GerritGroupMemberStatus{
			FailureCount: 2,
		},
	}

	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "SetFailureCount",
			args: args{
				fc: &instance,
			},
			want: getTimeout(2, 1*time.Second),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, SetFailureCount(tt.args.fc))
		})
	}
}

func Test_getTimeout(t *testing.T) {
	t.Parallel()

	type args struct {
		factor       int64
		baseDuration time.Duration
	}

	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "getTimeout with factor 1",
			args: args{
				factor:       1,
				baseDuration: 1 * time.Second,
			},
			want: time.Duration(float64(1*time.Second) * math.Pow(math.E, 2.0)),
		},
		{
			name: "getTimeout with factor 10 and exponential timeout > 1 hour",
			args: args{
				factor:       20,
				baseDuration: 1 * time.Second,
			},
			want: 1 * time.Hour,
		},
		{
			name: "getTimeout with factor 200 and exponential timeout < 0",
			args: args{
				factor:       200,
				baseDuration: 1 * time.Second,
			},
			want: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, getTimeout(tt.args.factor, tt.args.baseDuration))
		})
	}
}
