package main

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestServer_handleRequestSuccess(t *testing.T) {
	type fields struct {
		URL                       string
		RequestErrorThreshold     int
		HealthCheckErrorThreshold int
		SlowRequestThreshold      int
		Mutex                     sync.Mutex
		HitCount                  decimal.Decimal
		Weight                    int
		HealthCheckErrorCount     int
		RequestErrorCount         int
		SlowRequestCount          int
		Status                    ServerStatus
		RecoverTime               time.Time
		AverageLatency            decimal.Decimal
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success - normal path",
			fields: fields{
				RequestErrorCount: 10,
				Status:            ServerStatusUNHEALTHY,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				URL:                       tt.fields.URL,
				RequestErrorThreshold:     tt.fields.RequestErrorThreshold,
				HealthCheckErrorThreshold: tt.fields.HealthCheckErrorThreshold,
				SlowRequestThreshold:      tt.fields.SlowRequestThreshold,
				HitCount:                  tt.fields.HitCount,
				Weight:                    tt.fields.Weight,
				HealthCheckErrorCount:     tt.fields.HealthCheckErrorCount,
				RequestErrorCount:         tt.fields.RequestErrorCount,
				SlowRequestCount:          tt.fields.SlowRequestCount,
				Status:                    tt.fields.Status,
				RecoverTime:               tt.fields.RecoverTime,
				AverageLatency:            tt.fields.AverageLatency,
			}
			s.handleRequestSuccess()
			if s.RequestErrorCount != 0 {
				t.Errorf("field RequestErrorCount error, want: 0, got: %d", s.RequestErrorCount)
			}
			if s.Status != ServerStatusHEALTHY {
				t.Errorf("field Status error, want: HEALTHY, got: %s", s.Status)
			}
		})
	}
}

func TestServer_handleRequestError(t *testing.T) {
	type fields struct {
		URL                       string
		RequestErrorThreshold     int
		HealthCheckErrorThreshold int
		SlowRequestThreshold      int
		Mutex                     sync.Mutex
		HitCount                  decimal.Decimal
		Weight                    int
		HealthCheckErrorCount     int
		RequestErrorCount         int
		SlowRequestCount          int
		Status                    ServerStatus
		RecoverTime               time.Time
		AverageLatency            decimal.Decimal
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "succes - normal path",
			fields: fields{
				Status:                ServerStatusHEALTHY,
				RequestErrorCount:     2,
				RequestErrorThreshold: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				URL:                       tt.fields.URL,
				RequestErrorThreshold:     tt.fields.RequestErrorThreshold,
				HealthCheckErrorThreshold: tt.fields.HealthCheckErrorThreshold,
				SlowRequestThreshold:      tt.fields.SlowRequestThreshold,
				HitCount:                  tt.fields.HitCount,
				Weight:                    tt.fields.Weight,
				HealthCheckErrorCount:     tt.fields.HealthCheckErrorCount,
				RequestErrorCount:         tt.fields.RequestErrorCount,
				SlowRequestCount:          tt.fields.SlowRequestCount,
				Status:                    tt.fields.Status,
				RecoverTime:               tt.fields.RecoverTime,
				AverageLatency:            tt.fields.AverageLatency,
			}
			s.handleRequestError()
			if s.RequestErrorCount != 3 {
				t.Errorf("field RequestErrorCount error, want: 3, got: %d", s.RequestErrorCount)
			}
			if s.Status != ServerStatusUNHEALTHY {
				t.Errorf("field Status error, want: UNHEALTHY, got: %s", s.Status)
			}
			if s.RecoverTime.IsZero() {
				t.Errorf("field Status error, want: %s, got: nil", s.RecoverTime.String())
			}
		})
	}
}

func TestServer_handleHealthCheckSuccess(t *testing.T) {
	type fields struct {
		URL                       string
		RequestErrorThreshold     int
		HealthCheckErrorThreshold int
		SlowRequestThreshold      int
		Mutex                     sync.Mutex
		HitCount                  decimal.Decimal
		Weight                    int
		HealthCheckErrorCount     int
		RequestErrorCount         int
		SlowRequestCount          int
		Status                    ServerStatus
		RecoverTime               time.Time
		AverageLatency            decimal.Decimal
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "succes - normal path",
			fields: fields{
				Status:                    ServerStatusDOWN,
				HealthCheckErrorCount:     2,
				HealthCheckErrorThreshold: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				URL:                       tt.fields.URL,
				RequestErrorThreshold:     tt.fields.RequestErrorThreshold,
				HealthCheckErrorThreshold: tt.fields.HealthCheckErrorThreshold,
				SlowRequestThreshold:      tt.fields.SlowRequestThreshold,
				HitCount:                  tt.fields.HitCount,
				Weight:                    tt.fields.Weight,
				HealthCheckErrorCount:     tt.fields.HealthCheckErrorCount,
				RequestErrorCount:         tt.fields.RequestErrorCount,
				SlowRequestCount:          tt.fields.SlowRequestCount,
				Status:                    tt.fields.Status,
				RecoverTime:               tt.fields.RecoverTime,
				AverageLatency:            tt.fields.AverageLatency,
			}
			s.handleHealthCheckSuccess()
			if s.HealthCheckErrorCount != 0 {
				t.Errorf("field RequestErrorCount error, want: 0, got: %d", s.RequestErrorCount)
			}
			if s.Status != ServerStatusHEALTHY {
				t.Errorf("field Status error, want: HEALTHY, got: %s", s.Status)
			}
		})
	}
}

func TestServer_handleHealthCheckError(t *testing.T) {
	type fields struct {
		URL                       string
		RequestErrorThreshold     int
		HealthCheckErrorThreshold int
		SlowRequestThreshold      int
		Mutex                     sync.Mutex
		HitCount                  decimal.Decimal
		Weight                    int
		HealthCheckErrorCount     int
		RequestErrorCount         int
		SlowRequestCount          int
		Status                    ServerStatus
		RecoverTime               time.Time
		AverageLatency            decimal.Decimal
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success - normal path",
			fields: fields{
				Status:                    ServerStatusHEALTHY,
				HealthCheckErrorCount:     1,
				HealthCheckErrorThreshold: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				URL:                       tt.fields.URL,
				RequestErrorThreshold:     tt.fields.RequestErrorThreshold,
				HealthCheckErrorThreshold: tt.fields.HealthCheckErrorThreshold,
				SlowRequestThreshold:      tt.fields.SlowRequestThreshold,
				Mutex:                     tt.fields.Mutex,
				HitCount:                  tt.fields.HitCount,
				Weight:                    tt.fields.Weight,
				HealthCheckErrorCount:     tt.fields.HealthCheckErrorCount,
				RequestErrorCount:         tt.fields.RequestErrorCount,
				SlowRequestCount:          tt.fields.SlowRequestCount,
				Status:                    tt.fields.Status,
				RecoverTime:               tt.fields.RecoverTime,
				AverageLatency:            tt.fields.AverageLatency,
			}
			s.handleHealthCheckError()
			if s.HealthCheckErrorCount != 2 {
				t.Errorf("field RequestErrorCount error, want: 2, got: %d", s.RequestErrorCount)
			}
			if s.Status != ServerStatusDOWN {
				t.Errorf("field Status error, want: DOWN, got: %s", s.Status)
			}
		})
	}
}

func TestNewLoadBalancer(t *testing.T) {
	tests := []struct {
		name string
		want *LoadBalancer
	}{
		{
			name: "success - normal path",
			want: &LoadBalancer{
				Servers:         []*Server{},
				WeightCounter:   2,
				NormalWeight:    2,
				SlowWeight:      1,
				ExpectedLatency: decimal.NewFromInt(100),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLoadBalancer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLoadBalancer() = %v, want %v", got, tt.want)
			}
		})
	}
}
