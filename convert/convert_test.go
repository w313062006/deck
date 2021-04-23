package convert

import (
	"reflect"
	"testing"

	"github.com/kong/deck/file"
	"github.com/kong/deck/utils"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

func Test_validConversion(t *testing.T) {
	type args struct {
		from Format
		to   Format
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "empty from and to result in an error",
			want:    false,
			wantErr: true,
		},
		{
			name: "valid conversions return true",
			args: args{
				from: FormatKongGateway,
				to:   FormatKonnect,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid conversions return false",
			args: args{
				from: FormatKonnect,
				to:   FormatKongGateway,
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validConversion(tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("validConversion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validConversion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    Format
		wantErr bool
	}{
		{
			name: "parses valid values",
			args: args{
				key: "kong-gateway",
			},
			want:    FormatKongGateway,
			wantErr: false,
		},
		{
			name: "parses values in a case-insensitive manner",
			args: args{
				key: "koNNect",
			},
			want:    FormatKonnect,
			wantErr: false,
		},
		{
			name: "parse fails with invalid values",
			args: args{
				key: "k42",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kongServiceToKonnectServicePackage(t *testing.T) {
	type args struct {
		service file.FService
	}
	tests := []struct {
		name    string
		args    args
		want    file.FServicePackage
		wantErr bool
	}{
		{
			name: "converts a kong service to service package",
			args: args{
				service: file.FService{
					Service: kong.Service{
						Name: kong.String("foo"),
						Host: kong.String("foo.example.com"),
					},
				},
			},
			want: file.FServicePackage{
				Name:        kong.String("foo"),
				Description: kong.String("placeholder description for foo service package"),
				Versions: []file.FServiceVersion{
					{
						Version: kong.String("v1"),
						Implementation: &file.Implementation{
							Type: utils.ImplementationTypeKongGateway,
							Kong: &file.Kong{
								Service: &file.FService{
									Service: kong.Service{
										Host: kong.String("foo.example.com"),
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "converts fails for kong services without a name",
			args: args{
				service: file.FService{
					Service: kong.Service{
						ID:   kong.String("service-id"),
						Host: kong.String("foo.example.com"),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kongServiceToKonnectServicePackage(tt.args.service)
			if (err != nil) != tt.wantErr {
				t.Errorf("kongServiceToKonnectServicePackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got = zeroOutID(got)
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func zeroOutID(sp file.FServicePackage) file.FServicePackage {
	res := sp.DeepCopy()
	for _, v := range res.Versions {
		if v.Implementation != nil && v.Implementation.Kong != nil &&
			v.Implementation.Kong.Service != nil {
			v.Implementation.Kong.Service.ID = nil
		}
	}
	return *res
}

func Test_convertKongGatewayToKonnect(t *testing.T) {
	type args struct {
		inputFilename          string
		outputFilename         string
		expectedOutputFilename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "errors out when a nameless service is present in the input",
			args: args{
				inputFilename: "testdata/1/input.yaml",
			},
			wantErr: true,
		},
		{
			name: "errors out when input file doesn't exist",
			args: args{
				inputFilename: "testdata/1/input-does-not-exist.yaml",
			},
			wantErr: true,
		},
		{
			name: "converts from Kong Gateway to Konnect format",
			args: args{
				inputFilename:          "testdata/2/input.yaml",
				outputFilename:         "testdata/2/output.yaml",
				expectedOutputFilename: "testdata/2/output-expected.yaml",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := convertKongGatewayToKonnect(tt.args.inputFilename, tt.args.outputFilename)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertKongGatewayToKonnect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				got, err := file.GetContentFromFiles([]string{tt.args.outputFilename})
				if err != nil {
					t.Errorf("failed to read output file: %v", err)
				}
				want, err := file.GetContentFromFiles([]string{tt.args.expectedOutputFilename})
				if err != nil {
					t.Errorf("failed to read output file: %v", err)
				}
				if !equalContents(got, want) {
					assert.Equal(t, want, got)
				}
			}
		})
	}
}

func equalContents(got, want *file.Content) bool {
	var packages []file.FServicePackage
	for _, sp := range got.ServicePackages {
		sp := sp
		sp = zeroOutID(sp)
		packages = append(packages, sp)
	}
	got.ServicePackages = packages
	packages = nil
	for _, sp := range want.ServicePackages {
		sp := sp
		sp = zeroOutID(sp)
		packages = append(packages, sp)
	}
	want.ServicePackages = packages
	return reflect.DeepEqual(want, got)
}
