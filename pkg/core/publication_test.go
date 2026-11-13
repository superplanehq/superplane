package core

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPublicationOrderTieBreakers(t *testing.T) {
	now := time.Now()
	lower := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	higher := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	base := PublicationOrder{SourceRunID: lower, SourceCreatedAt: now, AttemptRootEventID: lower, AttemptCreatedAt: now}
	for _, test := range []struct {
		name   string
		change func(*PublicationOrder)
	}{
		{"source time wins over IDs and attempts", func(o *PublicationOrder) {
			o.SourceCreatedAt = now.Add(time.Second)
			o.AttemptCreatedAt = now.Add(-time.Hour)
		}},
		{"source ID wins over attempt time", func(o *PublicationOrder) { o.SourceRunID = higher; o.AttemptCreatedAt = now.Add(-time.Hour) }},
		{"attempt time breaks source ties", func(o *PublicationOrder) { o.AttemptCreatedAt = now.Add(time.Second) }},
		{"attempt ID breaks timestamp ties", func(o *PublicationOrder) { o.AttemptRootEventID = higher }},
	} {
		t.Run(test.name, func(t *testing.T) {
			newer := base
			test.change(&newer)
			require.Positive(t, newer.Compare(base))
			require.Negative(t, base.Compare(newer))
		})
	}
	require.Zero(t, base.Compare(base))
}
