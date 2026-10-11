export const mergeConfidenceAvailableIntegrations = [
  {
    name: "github",
    label: "GitHub",
    capabilities: [
      {
        type: "TYPE_TRIGGER",
        name: "github.onPullRequest",
        label: "On Pull Request",
        configuration: [
          { name: "customName", label: "Run title", type: "string" },
          { name: "repository", label: "Repository", type: "string", required: true },
          {
            name: "actions",
            label: "Actions",
            type: "multi-select",
            typeOptions: {
              multiSelect: {
                options: [
                  { label: "Labeled", value: "labeled" },
                  { label: "Assigned", value: "assigned" },
                ],
              },
            },
          },
          {
            name: "ignoreDrafts",
            label: "Ignore draft pull requests",
            type: "boolean",
            description: "Do not start a run when the pull request is a draft.",
          },
          {
            name: "onlyFactoryPullRequests",
            label: "Only pull requests in this factory",
            type: "boolean",
            description: "Start a run only when the pull request belongs to this factory.",
          },
        ],
      },
      {
        type: "TYPE_TRIGGER",
        name: "github.onIssue",
        label: "On Issue",
        configuration: [
          { name: "customName", label: "Run title", type: "string" },
          { name: "repository", label: "Repository", type: "string", required: true },
          {
            name: "actions",
            label: "Actions",
            type: "multi-select",
            typeOptions: {
              multiSelect: {
                options: [
                  { label: "Opened", value: "opened" },
                  { label: "Edited", value: "edited" },
                  { label: "Reopened", value: "reopened" },
                  { label: "Labeled", value: "labeled" },
                  { label: "Closed", value: "closed" },
                ],
              },
            },
          },
          {
            name: "labels",
            label: "Issue has one of these labels",
            type: "list",
            typeOptions: {
              list: {
                itemLabel: "Label",
                itemDefinition: { type: "string" },
              },
            },
          },
          { name: "labelFilterMode", label: "Label filter", type: "select" },
          { name: "assignment", label: "Assignment", type: "select" },
          { name: "authorsWithAccess", label: "Author is a repository collaborator", type: "boolean" },
          {
            name: "superplaneLabelAdded",
            label: 'The "superplane" label is added to the issue',
            type: "boolean",
          },
        ],
      },
    ],
  },
];
