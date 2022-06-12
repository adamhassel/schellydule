package shelly

import (
	"reflect"
	"testing"
)

func TestSchedules_FindMatching(t *testing.T) {
	type args struct {
		j JobSpec
	}
	tests := []struct {
		name    string
		s       Schedules
		args    args
		want    JobSpec
		wantErr bool
	}{
		{
			name: "trival, just one pair",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
			},

			want: JobSpec{
				Id:       1,
				Enable:   false,
				Timespec: "13 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "trival, just one pair, backwards",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
			},

			want: JobSpec{
				Id:       0,
				Enable:   true,
				Timespec: "12 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
			},

			want: JobSpec{
				Id:       1,
				Enable:   false,
				Timespec: "13 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
			},

			want: JobSpec{
				Id:       3,
				Enable:   false,
				Timespec: "15 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs, backwards",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
			},

			want: JobSpec{
				Id:       0,
				Enable:   true,
				Timespec: "12 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "almost trival, two pairs (but getting the other one), backwards",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},

			want: JobSpec{
				Id:       2,
				Enable:   true,
				Timespec: "14 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "arbitrary input, will still find a match",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       42,
					Enable:   true,
					Timespec: "9 * * * * *",
				},
			},

			want: JobSpec{
				Id:       1,
				Enable:   false,
				Timespec: "13 * * * * *",
			},
			wantErr: false,
		},
		{
			name: "arbitrary input, but no match",
			s: Schedules{
				JobSpec{
					Id:       0,
					Enable:   true,
					Timespec: "12 * * * * *",
				},
				JobSpec{
					Id:       1,
					Enable:   false,
					Timespec: "13 * * * * *",
				},
				JobSpec{
					Id:       2,
					Enable:   true,
					Timespec: "14 * * * * *",
				},
				JobSpec{
					Id:       3,
					Enable:   false,
					Timespec: "15 * * * * *",
				},
			},
			args: args{
				j: JobSpec{
					Id:       42,
					Enable:   true,
					Timespec: "16 * * * * *",
				},
			},

			want:    JobSpec{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.FindMatching(tt.args.j)
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
