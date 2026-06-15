package canvases

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type canvasChangeRequestsSerializationData struct {
	approvalsByRequestID map[uuid.UUID][]models.CanvasChangeRequestApproval
	usersByID            map[string]*models.User
	versionsByID         map[uuid.UUID]*models.CanvasVersion
}

func loadCanvasChangeRequestSerializationData(
	organizationID string,
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
) ([]models.CanvasChangeRequestApproval, map[string]*models.User, error) {
	approvals, err := models.ListCanvasChangeRequestApprovals(request.WorkflowID, request.ID)
	if err != nil {
		return nil, nil, err
	}

	approvalsByRequestID := map[uuid.UUID][]models.CanvasChangeRequestApproval{
		request.ID: approvals,
	}
	versionsByID := map[uuid.UUID]*models.CanvasVersion{}
	if version != nil {
		versionsByID[version.ID] = version
	}

	usersByID, err := usersByIDForCanvasChangeRequests(
		organizationID,
		[]models.CanvasChangeRequest{*request},
		approvalsByRequestID,
		versionsByID,
	)
	if err != nil {
		return nil, nil, err
	}

	return approvals, usersByID, nil
}

func loadCanvasChangeRequestsSerializationData(
	requests []models.CanvasChangeRequest,
	organizationID string,
) (*canvasChangeRequestsSerializationData, error) {
	if len(requests) == 0 {
		return &canvasChangeRequestsSerializationData{
			approvalsByRequestID: map[uuid.UUID][]models.CanvasChangeRequestApproval{},
			usersByID:            map[string]*models.User{},
			versionsByID:         map[uuid.UUID]*models.CanvasVersion{},
		}, nil
	}

	workflowID := requests[0].WorkflowID

	changeRequestIDs := make([]uuid.UUID, len(requests))
	versionIDSet := make(map[uuid.UUID]struct{}, len(requests))
	versionIDs := make([]uuid.UUID, 0, len(requests))
	for i := range requests {
		changeRequestIDs[i] = requests[i].ID
		if _, ok := versionIDSet[requests[i].VersionID]; ok {
			continue
		}
		versionIDSet[requests[i].VersionID] = struct{}{}
		versionIDs = append(versionIDs, requests[i].VersionID)
	}

	versionsByID, err := models.FindCanvasVersionsByIDs(workflowID, versionIDs)
	if err != nil {
		return nil, err
	}

	approvalsByRequestID, err := models.ListCanvasChangeRequestApprovalsByRequestIDs(workflowID, changeRequestIDs)
	if err != nil {
		return nil, err
	}

	usersByID, err := usersByIDForCanvasChangeRequests(organizationID, requests, approvalsByRequestID, versionsByID)
	if err != nil {
		return nil, err
	}

	return &canvasChangeRequestsSerializationData{
		approvalsByRequestID: approvalsByRequestID,
		usersByID:            usersByID,
		versionsByID:         versionsByID,
	}, nil
}

func usersByIDForCanvasChangeRequests(
	organizationID string,
	requests []models.CanvasChangeRequest,
	approvalsByRequestID map[uuid.UUID][]models.CanvasChangeRequestApproval,
	versionsByID map[uuid.UUID]*models.CanvasVersion,
) (map[string]*models.User, error) {
	idSet := make(map[string]struct{})
	for i := range requests {
		if requests[i].OwnerID != nil {
			idSet[requests[i].OwnerID.String()] = struct{}{}
		}
	}

	for _, approvals := range approvalsByRequestID {
		for _, approval := range approvals {
			if approval.ActorUserID != nil {
				idSet[approval.ActorUserID.String()] = struct{}{}
			}
		}
	}

	for _, version := range versionsByID {
		if version.OwnerID != nil {
			idSet[version.OwnerID.String()] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return map[string]*models.User{}, nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	users, err := models.FindUsersByIDsInOrganization(organizationID, ids)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[string]*models.User, len(users))
	for i := range users {
		usersByID[users[i].ID.String()] = &users[i]
	}

	return usersByID, nil
}

func versionForCanvasChangeRequestSerialization(
	data *canvasChangeRequestsSerializationData,
	request models.CanvasChangeRequest,
) (*models.CanvasVersion, error) {
	version := data.versionsByID[request.VersionID]
	if version == nil {
		return nil, status.Errorf(codes.Internal, "failed to load change request version: record not found")
	}

	return version, nil
}
