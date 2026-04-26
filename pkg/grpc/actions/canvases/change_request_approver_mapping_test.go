package canvases

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestParseCanvasChangeRequestApprovalConfigReturnsTrimmedApprovers(t *testing.T) {
	approvers, err := parseCanvasChangeRequestApprovalConfig(&pb.Canvas_ChangeManagement{
		Approvals: []*pb.Canvas_ChangeManagement_Approver{
			{
				Type:   pb.Canvas_ChangeManagement_Approver_TYPE_USER,
				UserId: "  7a88735c-7200-4e6a-a2af-957f71b9168f  ",
			},
			{
				Type:     pb.Canvas_ChangeManagement_Approver_TYPE_ROLE,
				RoleName: "  org_admin  ",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, []models.CanvasChangeRequestApprover{
		{Type: models.CanvasChangeRequestApproverTypeUser, User: "7a88735c-7200-4e6a-a2af-957f71b9168f"},
		{Type: models.CanvasChangeRequestApproverTypeRole, Role: "org_admin"},
	}, approvers)
}

func TestParseCanvasChangeRequestApprovalConfigRejectsDuplicateApprovers(t *testing.T) {
	_, err := parseCanvasChangeRequestApprovalConfig(&pb.Canvas_ChangeManagement{
		Approvals: []*pb.Canvas_ChangeManagement_Approver{
			{
				Type:     pb.Canvas_ChangeManagement_Approver_TYPE_ROLE,
				RoleName: "org_admin",
			},
			{
				Type:     pb.Canvas_ChangeManagement_Approver_TYPE_ROLE,
				RoleName: "org_admin",
			},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate role approver org_admin is not allowed")
}

func TestParseCanvasChangeRequestApprovalConfigRejectsNilApprover(t *testing.T) {
	_, err := parseCanvasChangeRequestApprovalConfig(&pb.Canvas_ChangeManagement{
		Approvals: []*pb.Canvas_ChangeManagement_Approver{nil},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "approver 1 is required")
}

func TestCanvasChangeRequestApproverTypeFromProtoRejectsUnknownType(t *testing.T) {
	_, err := canvasChangeRequestApproverTypeFromProto(pb.Canvas_ChangeManagement_Approver_Type(-1))

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported approver type")
}

func TestValidateCanvasChangeRequestApproverRejectsMissingFields(t *testing.T) {
	require.EqualError(
		t,
		validateCanvasChangeRequestApprover(models.CanvasChangeRequestApprover{
			Type: models.CanvasChangeRequestApproverTypeUser,
		}),
		"user approvers require user_id",
	)

	require.EqualError(
		t,
		validateCanvasChangeRequestApprover(models.CanvasChangeRequestApprover{
			Type: models.CanvasChangeRequestApproverTypeRole,
		}),
		"role approvers require role_name",
	)
}

func TestValidateCanvasChangeRequestApproversRejectsEmptyAndInvalidUUID(t *testing.T) {
	err := validateCanvasChangeRequestApprovers(nil, "org-123", nil)
	require.EqualError(t, err, "at least one approver is required")

	err = validateCanvasChangeRequestApprovers(nil, "org-123", []models.CanvasChangeRequestApprover{
		{
			Type: models.CanvasChangeRequestApproverTypeUser,
			User: "not-a-uuid",
		},
	})
	require.EqualError(t, err, "approver user not-a-uuid is not a valid UUID")
}
