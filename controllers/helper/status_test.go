package helper

import (
	"testing"
	"time"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/edp/v1"
)

func TestSetFailureCount(t *testing.T) {
	type args struct {
		fc FailureCountable
	}

	var instance gerritApi.GerritGroupMember = gerritApi.GerritGroupMember{
		Status: gerritApi.GerritGroupMemberStatus{
			FailureCount: 2,
		},
	}

	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{"SetFailureCount", args{&instance}, getTimeout(2, 10*time.Second)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetFailureCount(tt.args.fc); got != tt.want {
				t.Errorf("SetFailureCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
