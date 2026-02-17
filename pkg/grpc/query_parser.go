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

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		decodedVals := make([]string, 0, len(vals))
		for _, val := range vals {
			decoded, err := url.QueryUnescape(val)
			if err != nil {
				decoded = val
			}
			decodedVals = append(decodedVals, decoded)
		}
		if len(decodedVals) == 1 {
			parameters[key] = decodedVals[0]
			continue
		}
		parameters[key] = strings.Join(decodedVals, ",")
	}

	r.Parameters = parameters
	return nil
}
