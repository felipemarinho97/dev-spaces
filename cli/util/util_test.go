package util

import (
	"reflect"
	"testing"
)

func TestParseInstanceSpec(t *testing.T) {
	type args struct {
		spec string
	}
	tests := []struct {
		name    string
		args    args
		want    InstanceSpec
		wantErr bool
	}{
		{
			name: "the spec is just the instance type",
			args: args{
				spec: "t2.micro",
			},
			want: InstanceSpec{
				InstanceType: "t2.micro",
			},
			wantErr: false,
		},
		{
			name: "the spec is just the instance type on filter format",
			args: args{
				spec: "type:t2.micro",
			},
			want: InstanceSpec{
				InstanceType: "t2.micro",
			},
			wantErr: false,
		},
		{
			name: "the spec is just the instance type on filter format with spaces",
			args: args{
				spec: "type: t2.micro",
			},
			want: InstanceSpec{
				InstanceType: "t2.micro",
			},
			wantErr: false,
		},
		{
			name: "the spec is just the instance type on filter format with spaces and other filters",
			args: args{
				spec: "type: t2.micro, mem: 0.5, cpus: 1",
			},
			want: InstanceSpec{
				InstanceType: "t2.micro",
				MinMemory:    512,
				MinCPU:       1,
			},
			wantErr: false,
		},
		{
			name: "the spec is the cpu and memory",
			args: args{
				spec: "mem:0.5,cpus:1",
			},
			want: InstanceSpec{
				MinMemory: 512,
				MinCPU:    1,
			},
			wantErr: false,
		},
		{
			name: "the spec is the cpu and memory is an integer",
			args: args{
				spec: "mem:1,cpus:1",
			},
			want: InstanceSpec{
				MinMemory: 1024,
				MinCPU:    1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInstanceSpec(tt.args.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInstanceSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInstanceSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
