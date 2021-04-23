package convert

import (
	"fmt"
	"strings"

	"github.com/kong/deck/file"
	"github.com/kong/deck/utils"
	"github.com/kong/go-kong/kong"
)

type Format string

const (
	FormatKongGateway Format = "kong-gateway"
	FormatKonnect     Format = "konnect"
)

var (
	AllFormats = []Format{FormatKongGateway, FormatKonnect}

	conversions = map[Format]map[Format]bool{
		FormatKongGateway: {
			FormatKonnect: true,
		},
	}
)

func validConversion(from, to Format) (bool, error) {
	if fromMap, ok := conversions[from]; ok {
		if _, ok := fromMap[to]; ok {
			return ok, nil
		}
	}

	return false, fmt.Errorf("cannot convert from '%s' to '%s' format", from, to)
}

func ParseFormat(key string) (Format, error) {
	format := Format(strings.ToLower(key))
	switch format {
	case FormatKongGateway:
		return FormatKongGateway, nil
	case FormatKonnect:
		return FormatKonnect, nil
	default:
		return "", fmt.Errorf("invalid format: '%v'", key)
	}
}

func Convert(inputFilename, outputFilename string, from, to Format) error {
	if valid, err := validConversion(from, to); !valid {
		return err
	}

	switch {
	case from == FormatKongGateway && to == FormatKonnect:
		return convertKongGatewayToKonnect(inputFilename, outputFilename)
	default:
		return fmt.Errorf("cannot convert from '%s' to '%s' format", from, to)
	}
}

func convertKongGatewayToKonnect(inputFilename, outputFilename string) error {
	inputContent, err := file.GetContentFromFiles([]string{inputFilename})
	if err != nil {
		return err
	}

	outputContent := inputContent.DeepCopy()

	for _, service := range outputContent.Services {
		servicePackage, err := kongServiceToKonnectServicePackage(service)
		if err != nil {
			return err
		}
		outputContent.ServicePackages = append(outputContent.ServicePackages, servicePackage)
	}
	// Remove Kong Services from the file because all of them have been converted
	// into Service packages
	outputContent.Services = nil

	// all other entities are left as is

	if err := file.WriteContentToFile(outputContent, outputFilename, file.YAML); err != nil {
		return err
	}
	return nil
}

func kongServiceToKonnectServicePackage(service file.FService) (file.FServicePackage, error) {
	if service.Name == nil {
		return file.FServicePackage{}, fmt.Errorf("kong service with id '%s' doesn't have a name,"+
			"all services must be named to convert them from %s to %s format",
			*service.ID, FormatKongGateway, FormatKonnect)
	}

	serviceName := *service.Name
	// Kong service MUST contain an ID and no name in Konnect representation
	serviceCopy := service.DeepCopy()
	serviceCopy.Name = nil
	serviceCopy.ID = kong.String(utils.UUID())

	// convert Kong Service to a Service Package
	return file.FServicePackage{
		Name:        &serviceName,
		Description: kong.String("placeholder description for " + serviceName + " service package"),
		Versions: []file.FServiceVersion{
			{
				Version: kong.String("v1"),
				Implementation: &file.Implementation{
					Type: utils.ImplementationTypeKongGateway,
					Kong: &file.Kong{
						Service: serviceCopy,
					},
				},
			},
		},
	}, nil
}
