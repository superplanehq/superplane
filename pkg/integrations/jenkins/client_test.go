package jenkins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Client__EncodeJobPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple job",
			input:    "my-job",
			expected: "job/my-job",
		},
		{
			name:     "job in folder",
			input:    "folder/my-job",
			expected: "job/folder/job/my-job",
		},
		{
			name:     "deeply nested job",
			input:    "a/b/c",
			expected: "job/a/job/b/job/c",
		},
		{
			name:     "job with spaces",
			input:    "my folder/my job",
			expected: "job/my%20folder/job/my%20job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeJobPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test__Client__ParseQueueID(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		expectedID  int64
		expectError bool
	}{
		{
			name:       "standard location",
			location:   "https://jenkins.example.com/queue/item/42/",
			expectedID: 42,
		},
		{
			name:       "without trailing slash",
			location:   "https://jenkins.example.com/queue/item/42",
			expectedID: 42,
		},
		{
			name:        "invalid id",
			location:    "https://jenkins.example.com/queue/item/abc/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parseQueueID(tt.location)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func Test__Client__FlattenJobs(t *testing.T) {
	t.Run("flat list", func(t *testing.T) {
		jobs := []Job{
			{Name: "job1", FullName: "job1"},
			{Name: "job2", FullName: "job2"},
		}

		result := flattenJobs(jobs)
		assert.Len(t, result, 2)
	})

	t.Run("nested folders", func(t *testing.T) {
		jobs := []Job{
			{
				Name:     "folder1",
				FullName: "folder1",
				Jobs: []Job{
					{Name: "job1", FullName: "folder1/job1"},
					{Name: "job2", FullName: "folder1/job2"},
				},
			},
			{Name: "job3", FullName: "job3"},
		}

		result := flattenJobs(jobs)
		assert.Len(t, result, 3)
		assert.Equal(t, "folder1/job1", result[0].FullName)
		assert.Equal(t, "folder1/job2", result[1].FullName)
		assert.Equal(t, "job3", result[2].FullName)
	})

	t.Run("empty list", func(t *testing.T) {
		result := flattenJobs([]Job{})
		assert.Len(t, result, 0)
	})
}
