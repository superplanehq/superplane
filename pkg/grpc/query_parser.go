package grpc

import (
	"net/url"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/protobuf/proto"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

type QueryParser struct{}

func (p *QueryParser) Parse(target proto.Message, values url.Values, filter *utilities.DoubleArray) error {
	switch req := target.(type) {
	case *pb.ListIntegrationResourcesRequest:
		return populateListIntegrationResourcesParams(values, req)
	}

	defaultParser := runtime.DefaultQueryParser{}
	return defaultParser.Parse(target, values, filter)
}

func populateListIntegrationResourcesParams(values url.Values, r *pb.ListIntegrationResourcesRequest) error {
	parameters := map[string]string{}

	encodedParameters := values.Encode()
	queryParams := strings.Split(encodedParameters, "&")
	for _, queryParam := range queryParams {
		parts := strings.Split(queryParam, "=")
		if len(parts) == 2 {
			parameters[parts[0]] = parts[1]
		}
	}

	r.Parameters = parameters
	return nil
}
