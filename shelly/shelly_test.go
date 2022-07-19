package shelly

import (
	"reflect"
	"testing"
	"time"

	"github.com/adamhassel/schedule"
)

type lt struct {
	time.Time
}

func (t *lt) incr(h time.Duration) time.Time {
	t.Time = t.Time.Add(h * time.Hour)
	return t.Time
}
func (t *lt) n() time.Time {
	return t.Time
}

func TestShellySchedule(t *testing.T) {
	type args struct {
		in     schedule.Schedule
		enable bool
	}
	tm := lt{time.Now().Truncate(time.Hour)}
	in := schedule.Schedule{
		{
			Start: tm.Time,
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
		{
			Start: tm.incr(1),
			Stop:  tm.incr(1),
			Cost:  0,
		},
	}
	tests := []struct {
		name string
		args args
		want Schedule
	}{
		{
			name: "check size",
			args: args{
				in:     in,
				enable: true,
			},
			want: Schedule{
				Jobs: []JobSpec{
					{
						Id:       0,
						Enable:   false,
						Timespec: "",
						Calls:    nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShellySchedule(tt.args.in, tt.args.enable); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ShellySchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}
