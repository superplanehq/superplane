import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@/components/ui/button";

import { SpotlightPanelEditor } from "./SpotlightPanelEditor";
import { DEFAULT_SPOTLIGHT_CONTENT, type SpotlightPanelContent } from "./spotlightContent";

/**
 * Ground-up edit experience for the spotlight panel — a self-contained replica
 * of the real panel editor modal (header, Form/YAML tabs, Save/Cancel) with an
 * always-on live preview.
 *
 * Runs are the primary source: the sample records are shaped like the `runs`
 * rows `useWidgetData` produces (derived `status`/`nodeName`/`createdAt`/
 * `durationMs` plus a raw `executions` stage array), grounded in two real
 * SuperPlane apps — the "SuperPlane SaaS" delivery pipeline and the "Docs
 * Reviewer" PR agent. `LatestRun` shows the zero-config default; `DocsReview`
 * remaps the slots onto the run's payload.
 */
const meta = {
  title: "Console/Spotlight Editor (prototype)",
  component: SpotlightPanelEditor,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof SpotlightPanelEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

const avatar = (user: string) => `https://github.com/${user}.png`;

function EditorHarness({ initialContent, sampleRow }: { initialContent: SpotlightPanelContent; sampleRow: unknown }) {
  const [open, setOpen] = useState(true);
  const [content, setContent] = useState<SpotlightPanelContent>(initialContent);
  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-8 dark:bg-gray-950">
      <Button type="button" onClick={() => setOpen(true)}>
        Open spotlight editor
      </Button>
      <SpotlightPanelEditor
        open={open}
        onOpenChange={setOpen}
        initialContent={content}
        onSave={(next) => setContent(next)}
        sampleRow={sampleRow}
      />
    </div>
  );
}

/**
 * SuperPlane SaaS — the latest run of the delivery pipeline, shaped like a
 * `runs` data source row: derived `status`/`nodeName`/`durationMs` plus the raw
 * `executions` array the banner reads as its stage strip. The last stage is
 * still in flight (`RESULT_NONE` + `STATE_STARTED`), so it renders as running.
 */
const latestRunRow = {
  status: "passed",
  nodeName: "Deploy Helm — Production",
  createdAt: new Date(Date.now() - 18 * 60 * 1000).toISOString(),
  finishedAt: new Date(Date.now() - 17 * 60 * 1000).toISOString(),
  durationMs: 26 * 1000,
  rootEvent: { customName: "superplanehq/launchpad · main" },
  executions: [
    { nodeName: "SaaS CI", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Build Image", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Deploy Helm — Staging", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Deploy Helm — Production", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Promote — Production", state: "STATE_STARTED", result: "RESULT_NONE" },
  ],
};

/** The zero-config default: a runs source resolves every slot straight out of the box. */
export const LatestRun: Story = {
  render: () => <EditorHarness sampleRow={latestRunRow} initialContent={DEFAULT_SPOTLIGHT_CONTENT} />,
};

/**
 * Docs Reviewer — a different app on the same runs source, with the slots
 * remapped onto the run's `payload` (PR title/author) while the stage strip
 * still reads the run's `executions`. One stage failed and the next is running.
 */
const docsReviewRow = {
  status: "failed",
  nodeName: "Docs Reviewer",
  createdAt: new Date(Date.now() - 50 * 60 * 1000).toISOString(),
  durationMs: 3 * 1000,
  rootEvent: { customName: "superplanehq/superplane · #6010" },
  payload: {
    pull_request: {
      title: "fix: refine run inspector input timeline behavior",
      html_url: "https://github.com/superplanehq/superplane/pull/6010",
    },
    sender: { login: "forestileao", avatar_url: avatar("forestileao") },
  },
  agent: { name: "SuperPlane Docs Agent", avatar_url: avatar("superplanehq") },
  executions: [
    { nodeName: "Skip if Draft", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Mark Pending", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Analyze Docs", state: "STATE_FINISHED", result: "RESULT_FAILED" },
    { nodeName: "Create Docs Issue", state: "STATE_STARTED", result: "RESULT_NONE" },
  ],
};

export const DocsReview: Story = {
  render: () => (
    <EditorHarness
      sampleRow={docsReviewRow}
      initialContent={{
        ...DEFAULT_SPOTLIGHT_CONTENT,
        kicker: "Docs review",
        statusField: "status",
        statusLabelField: "status",
        actorNameField: "payload.sender.login",
        actorAvatarField: "payload.sender.avatar_url",
        titleField: "payload.pull_request.title",
        hrefField: "payload.pull_request.html_url",
        subtitleField: "rootEvent.customName",
        approverNameField: "agent.name",
        approverAvatarField: "agent.avatar_url",
        approverLabel: "Reviewed by",
        checksField: "executions",
        checkNameField: "nodeName",
        checkStatusField: "result",
      }}
    />
  ),
};

/** Invalid — no title or actor mapped, so the banner has no headline. */
export const Invalid: Story = {
  render: () => (
    <EditorHarness
      sampleRow={latestRunRow}
      initialContent={{ ...DEFAULT_SPOTLIGHT_CONTENT, titleField: "", actorNameField: "" }}
    />
  ),
};
