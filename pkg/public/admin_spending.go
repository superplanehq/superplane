package public

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) adminGetOrganizationSpendingReport(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	req, err := spendingReportRequestFromQuery(r, orgID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := organizations.DescribeOrganizationSpendingReport(r.Context(), orgID.String(), req)
	if err != nil {
		writeAdminSpendingReportError(w, err)
		return
	}

	body, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(response)
	if err != nil {
		log.Errorf("admin: failed to encode organization spending report: %v", err)
		http.Error(w, "Failed to load organization spending", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func spendingReportRequestFromQuery(r *http.Request, orgID string) (*pb.DescribeOrganizationSpendingReportRequest, error) {
	query := r.URL.Query()
	startTime, err := parseSpendingReportTime(query.Get("startTime"))
	if err != nil {
		return nil, fmt.Errorf("invalid startTime")
	}
	endTime, err := parseSpendingReportTime(query.Get("endTime"))
	if err != nil {
		return nil, fmt.Errorf("invalid endTime")
	}

	return &pb.DescribeOrganizationSpendingReportRequest{
		Id:            orgID,
		StartTime:     startTime,
		EndTime:       endTime,
		FactoryId:     query.Get("factoryId"),
		Model:         query.Get("model"),
		MachineType:   query.Get("machineType"),
		TaskOwnerId:   query.Get("taskOwnerId"),
		FundingSource: query.Get("fundingSource"),
		GroupBy:       query.Get("groupBy"),
		TimeGrain:     query.Get("timeGrain"),
		UsageKind:     query.Get("usageKind"),
	}, nil
}

func parseSpendingReportTime(raw string) (*timestamppb.Timestamp, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(parsed), nil
}

func writeAdminSpendingReportError(w http.ResponseWriter, err error) {
	statusCode := runtime.HTTPStatusFromCode(grpcerrors.Code(err))
	if statusCode == 0 || statusCode == http.StatusOK {
		statusCode = http.StatusInternalServerError
	}

	message := grpcerrors.StatusMessage(err)
	if message == "" {
		message = "Failed to load organization spending"
	}
	http.Error(w, message, statusCode)
}
