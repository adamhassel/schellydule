package schellydule

import (
	"reflect"
	"testing"

	"github.com/adamhassel/schellydule/shelly"
)

func TestSchedules_FindMatching(t *testing.T) {
	type args struct {
		j shelly.JobSpec
	}
	var callOn = []shelly.Call{{
		Method: "switch.set",
		Params: map[string]interface{}{
			"on": true,
		},
	}}
	var callOff = []shelly.Call{{
		Method: "switch.set",
		Params: map[string]interface{}{
			"on": false,
		},
	}}
	tests := []struct {
		name    string
		s       shelly.Schedules
		args    args
		want    shelly.JobSpec
		wantErr bool
	}{
		{
			name: "trival, just one pair",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
			},

			want: shelly.JobSpec{
				Id:       1,
				Enable:   true,
				Timespec: "0 0 13 * * *",
				Calls:    callOff,
			},
			wantErr: false,
		},
		{
			name: "trival, just one pair, backwards",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
			},

			want: shelly.JobSpec{
				Id:       0,
				Enable:   true,
				Timespec: "0 0 12 * * *",
				Calls:    callOn,
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
			},

			want: shelly.JobSpec{
				Id:       1,
				Enable:   true,
				Timespec: "0 0 13 * * *",
				Calls:    callOff,
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
			},

			want: shelly.JobSpec{
				Id:       3,
				Enable:   true,
				Timespec: "0 0 15 * * *",
				Calls:    callOff,
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs, backwards",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
			},

			want: shelly.JobSpec{
				Id:       0,
				Enable:   true,
				Timespec: "0 0 12 * * *",
				Calls:    callOn,
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs (but getting the other one), backwards",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},

			want: shelly.JobSpec{
				Id:       2,
				Enable:   true,
				Timespec: "0 0 14 * * *",
				Calls:    callOn,
			},
			wantErr: false,
		},
		{
			name: "arbitrary input, will still find a match",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       42,
					Enable:   true,
					Timespec: "0 0 9 * * *",
					Calls:    callOn,
				},
			},

			want: shelly.JobSpec{
				Id:       1,
				Enable:   true,
				Timespec: "0 0 13 * * *",
				Calls:    callOff,
			},
			wantErr: false,
		},
		{
			name: "arbitrary input, but no match",
			s: shelly.Schedules{
				shelly.JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "0 0 12 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       1,
					Enable:   true,
					Timespec: "0 0 13 * * *",
					Calls:    callOff,
				},
				shelly.JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "0 0 14 * * *",
					Calls:    callOn,
				},
				shelly.JobSpec{
					Id:       3,
					Enable:   true,
					Timespec: "0 0 15 * * *",
					Calls:    callOff,
				},
			},
			args: args{
				j: shelly.JobSpec{
					Id:       42,
					Enable:   true,
					Timespec: "0 0 16 * * *",
					Calls:    callOn,
				},
			},

			want:    shelly.JobSpec{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindMatching(tt.args.j, tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindMatching() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindMatching() got = %v, want %v", got, tt.want)
			}
		})
	}
}
