import { materializeConsoleSpec } from "../lib/workflow-spec-files";

import capturedFixture from "./canvasAppResponses.json";
import softwareFactoryHowItWorks from "./repository/softwareFactory.howItWorks.md?raw";
import softwareFactoryReadme from "./repository/softwareFactory.README.md?raw";
import type { CanvasAppFixture } from "./handlers";

// Wired against the same Software Factory capture the LiveCanvas story uses,
// so the panels resolve to real triggers/runs. Splitting them across pages
// lets Storybook exercise the multi-page tab strip without needing a fresh
// capture.
const overviewReadme = `# Software Factory

A three-page walk-through of the multi-page console.

- **Overview** — this page. Landing and pipeline health.
- **Pipeline** — kanban board of in-flight PRs.
- **Playbook** — how the factory works, end to end.

Switch pages via the tabs at the top of the console.`;

const overviewMarkdownBody = `${overviewReadme}\n\n---\n\n${softwareFactoryReadme}`;

const runsDataSource = {
  kind: "runs" as const,
  limit: 100,
  triggers: ["on-issue-labeled-trigger", "component-node-4m9qti"],
};

const boardGroupBy = `{{ status == "passed" ? "Done" :
   status == "failed" || status == "cancelled" ? "Failed" :
   ("Mark PR Ready" in $) ? "Human review" :
   "In progress" }}`;

const boardWhere = [{ field: '$["Open Draft PR"].state', op: "exists" }];

const boardCard = {
  titleField: `{{ $["Open Draft PR"].data.title != null
   ? $["Open Draft PR"].data.title
   : payload.data.issue.title }}`,
  fields: [
    {
      field: '$["Open Draft PR"].data.number',
      format: "link",
      href: '{{ $["Open Draft PR"].data.html_url }}',
      label: "PR",
    },
    {
      field: "payload.data.issue.number",
      format: "link",
      href: "{{ payload.data.issue.html_url }}",
      label: "Issue",
    },
    { field: "durationMs", format: "duration", label: "Elapsed" },
    { field: "updatedAt", format: "relative", label: "Updated" },
  ],
};

const capturedApp = capturedFixture as CanvasAppFixture;

const multiPageConsoleYaml = materializeConsoleSpec({
  canvasId: capturedApp.canvasId,
  canvasName: capturedApp.canvas?.canvas?.metadata?.name ?? "Software Factory",
  pages: [
    {
      id: "overview",
      name: "Overview",
      panels: [
        {
          id: "submit-task",
          type: "nodes",
          content: {
            title: "Create a task",
            nodes: [
              {
                node: "create-task-start",
                formMode: "inline",
                showFieldLabels: false,
                showNodeLabel: false,
                showRun: true,
                submitLabel: "Work on it",
                triggerName: "Create Task",
              },
            ],
          },
        },
        {
          id: "overview-readme",
          type: "markdown",
          content: {
            title: "About this console",
            body: overviewMarkdownBody,
            variables: [],
          },
        },
        {
          id: "runs-total",
          type: "number",
          content: {
            title: "Runs (all time)",
            dataSource: runsDataSource,
            render: {
              kind: "number",
              aggregation: "count",
            },
          },
        },
        {
          id: "recent-runs",
          type: "table",
          content: {
            title: "Recent runs",
            dataSource: runsDataSource,
            render: {
              kind: "table",
              columns: [
                { field: "status", label: "Status", format: "status" },
                {
                  field: '$["Open Draft PR"].data.title',
                  label: "PR title",
                },
                {
                  field: '$["Open Draft PR"].data.number',
                  label: "PR",
                  format: "link",
                  href: '{{ $["Open Draft PR"].data.html_url }}',
                },
                { field: "durationMs", label: "Elapsed", format: "duration" },
                { field: "updatedAt", label: "Updated", format: "relative" },
              ],
              sort: { field: "updatedAt", order: "desc" },
            },
          },
        },
      ],
      layout: [
        { i: "submit-task", x: 0, y: 0, w: 4, h: 7, minW: 2, minH: 4 },
        { i: "overview-readme", x: 4, y: 0, w: 8, h: 7, minW: 4, minH: 3 },
        { i: "runs-total", x: 0, y: 7, w: 4, h: 4, minW: 2, minH: 3 },
        { i: "recent-runs", x: 4, y: 7, w: 8, h: 6, minW: 4, minH: 4 },
      ],
    },
    {
      id: "pipeline",
      name: "Pipeline",
      panels: [
        {
          id: "pipeline-board",
          type: "board",
          content: {
            title: "Factory pipeline",
            dataSource: runsDataSource,
            render: {
              kind: "board",
              groupBy: boardGroupBy,
              lanes: [
                { value: "In progress", color: "blue" },
                { value: "Human review", color: "yellow", label: "Human review" },
                { value: "Failed", color: "red" },
                { value: "Done", color: "green" },
              ],
              otherLane: false,
              sort: { field: "updatedAt", order: "desc" },
              where: boardWhere,
              emptyMessage: "No factory pull requests yet. Submit a task to start one.",
              card: boardCard,
            },
          },
        },
      ],
      layout: [{ i: "pipeline-board", x: 0, y: 0, w: 12, h: 14, minW: 6, minH: 6 }],
    },
    {
      id: "playbook",
      name: "Playbook",
      panels: [
        {
          id: "how-it-works",
          type: "markdown",
          content: {
            title: "How it works",
            body: softwareFactoryHowItWorks,
            variables: [],
          },
        },
      ],
      layout: [{ i: "how-it-works", x: 0, y: 0, w: 12, h: 12, minW: 4, minH: 3 }],
    },
  ],
});

// A multi-page variant of the default Software Factory fixture. Reuses every
// captured endpoint (canvas, versions, runs, memory, integrations) so the
// panels resolve to real data, but ships a three-page console.yaml so the
// tab strip renders with something meaningful in every tab.
export const multiPageConsoleFixture = {
  ...capturedApp,
  consoleYaml: multiPageConsoleYaml,
  repositoryFileContents: {
    "README.md": softwareFactoryReadme,
    ...capturedApp.repositoryFileContents,
  },
} satisfies CanvasAppFixture;
