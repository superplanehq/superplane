import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@/components/ui/button";

import { TablePanelEditor } from "./TablePanelEditor";
import { MockConsoleProvider } from "./__stories__/storyDecorators";
import {
  baseTableRender,
  columnFormatShowcaseRender,
  columnFormatShowcaseRows,
  executionRows,
  prRiskCheckRows,
  prRiskChecksTableRender,
} from "./__stories__/storyFixtures";
import type { TablePanelContent } from "./panelTypes";

/**
 * Edit experience for the table panel — the real panel editor modal (Form/YAML
 * tabs, validation, YAML diff) with an always-on live preview injected above
 * the real `TablePanelForm`.
 *
 * The preview is fed static sample rows (the mockup stand-in for `useWidgetData`
 * rows), and the stories cover each data source: runs, executions, and memory.
 */
const meta = {
  title: "Console/Table Editor (prototype)",
  component: TablePanelEditor,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof TablePanelEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

function EditorHarness({
  initialContent,
  sampleRows,
}: {
  initialContent: TablePanelContent;
  sampleRows: Record<string, unknown>[];
}) {
  const [open, setOpen] = useState(true);
  const [content, setContent] = useState<TablePanelContent>(initialContent);
  return (
    <MockConsoleProvider>
      <div className="flex min-h-screen items-center justify-center bg-slate-100 p-8 dark:bg-gray-950">
        <Button type="button" onClick={() => setOpen(true)}>
          Open table editor
        </Button>
        <TablePanelEditor
          open={open}
          onOpenChange={setOpen}
          initialContent={content}
          onSave={(next) => setContent(next)}
          sampleRows={sampleRows}
        />
      </div>
    </MockConsoleProvider>
  );
}

/** Runs — every column format (incl. the progress bar) plus a row action. */
export const AllColumnFormats: Story = {
  render: () => (
    <EditorHarness
      sampleRows={columnFormatShowcaseRows}
      initialContent={{
        title: "Column format showcase",
        dataSource: { kind: "runs", limit: 30 },
        render: columnFormatShowcaseRender,
      }}
    />
  ),
};

/** Executions — the default status / duration / relative-time table. */
export const RecentExecutions: Story = {
  render: () => (
    <EditorHarness
      sampleRows={executionRows}
      initialContent={{
        title: "Recent executions",
        dataSource: { kind: "executions", limit: 50 },
        render: baseTableRender,
      }}
    />
  ),
};

/** Memory — the `pr-risk-review` checks table, with a link column and a row action. */
export const PrRiskChecks: Story = {
  render: () => (
    <EditorHarness
      sampleRows={prRiskCheckRows}
      initialContent={{
        title: "Recent checks",
        dataSource: { kind: "memory", namespace: "checks" },
        render: prRiskChecksTableRender,
      }}
    />
  ),
};
