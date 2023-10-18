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

func TestParseAMIFilter(t *testing.T) {
	type args struct {
		filter string
	}
	tests := []struct {
		name    string
		args    args
		want    AMIFilter
		wantErr bool
	}{
		{
			name: "the filter is just the ami id",
			args: args{
				filter: "ami-0df435f331839b2d6",
			},
			want: AMIFilter{
				ID: "ami-0df435f331839b2d6",
			},
			wantErr: false,
		},
		{
			name: "the filter is just the ami id on filter format",
			args: args{
				filter: "id:ami-0df435f331839b2d6",
			},
			want: AMIFilter{
				ID: "ami-0df435f331839b2d6",
			},
			wantErr: false,
		},
		{
			name: "the filter is name and arch",
			args: args{
				filter: "name:my-ami,arch:x86_64",
			},
			want: AMIFilter{
				Name: "my-ami",
				Arch: "x86_64",
			},
			wantErr: false,
		},
		{
			name: "the filter is name and owner",
			args: args{
				filter: "name:my-ami,owner:123456789012",
			},
			want: AMIFilter{
				Name:  "my-ami",
				Owner: "123456789012",
			},
			wantErr: false,
		},
		{
			name: "the filter is name with wildcards and owner as fixed string",
			args: args{
				filter: "name:my-ami*,owner:amazon",
			},
			want: AMIFilter{
				Name:  "my-ami*",
				Owner: "amazon",
			},
			wantErr: false,
		},
		{
			name: "the filter is id and name",
			args: args{
				filter: "id:ami-0df435f331839b2d6,name:my-ami",
			},
			want: AMIFilter{
				ID:   "ami-0df435f331839b2d6",
				Name: "my-ami",
			},
			wantErr: false,
		},
		{
			name: "the filter is id and owner",
			args: args{
				filter: "id:ami-0df435f331839b2d6,owner:123456789012",
			},
			want: AMIFilter{
				ID:    "ami-0df435f331839b2d6",
				Owner: "123456789012",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAMIFilter(tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAMIFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAMIFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
