package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListFactories_IncludesLines(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	withLine, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := withLine.CreateLine(db, "plan-and-implement", nil)
	require.NoError(t, err)

	empty, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := ListFactories(ctx, r.Organization.ID.String())
	require.NoError(t, err)

	listedWithLine := listedFactoryByID(resp.Factories, withLine.ID.String())
	require.NotNil(t, listedWithLine)
	require.Len(t, listedWithLine.Lines, 1)
	assert.Equal(t, line.ID.String(), listedWithLine.Lines[0].Id)
	assert.Equal(t, "plan-and-implement", listedWithLine.Lines[0].Name)
	assert.Nil(t, listedWithLine.Lines[0].Metrics)

	listedEmpty := listedFactoryByID(resp.Factories, empty.ID.String())
	require.NotNil(t, listedEmpty)
	assert.Empty(t, listedEmpty.Lines)
}

func listedFactoryByID(factories []*pb.Factory, id string) *pb.Factory {
	for _, factory := range factories {
		if factory.GetId() == id {
			return factory
		}
	}
	return nil
}
