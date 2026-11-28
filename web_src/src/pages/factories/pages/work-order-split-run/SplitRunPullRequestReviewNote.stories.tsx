import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState, type ReactNode } from "react";

import type { FactoriesFactoryPullRequestMergeability } from "@/api-client";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const meta = {
  title: "Factories/Pages/Task Split Run/Pull request review strip",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story, context) => (
      <StoryQueryClient key={context.id}>
        <ComponentStoryShell className="min-h-svh bg-muted/30 p-6">
          <Story />
        </ComponentStoryShell>
      </StoryQueryClient>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

function StoryQueryClient({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function PopupShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[420px] w-full max-w-[750px] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-sm">
      <header className="flex shrink-0 items-center gap-3 border-b border-border px-5 py-3">
        <h2 className="min-w-0 flex-1 truncate text-[16px] font-semibold tracking-[-0.02em]">
          {OPEN_WORK_ORDER.title}
        </h2>
      </header>
      <div className="min-h-0 flex-1 px-5 py-4 text-[13px] text-muted-foreground">Automations log.</div>
      {children}
    </div>
  );
}

const footer = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER).footer;

export const Default: Story = {
  name: "Pull request is ready",
  render: () => (
    <PopupShell>
      {footer.note ? (
        <SplitRunAttentionNote note={footer.note} tone="waiting" actions={footer.actions} onAction={() => {}} />
      ) : null}
    </PopupShell>
  ),
};

export const ReadOnly: Story = {
  name: "Pull request is ready — no close actions",
  render: () => (
    <PopupShell>
      {footer.note ? <SplitRunAttentionNote note={footer.note} tone="waiting" actions={[]} /> : null}
    </PopupShell>
  ),
};

function statusParameters(mergeability: FactoriesFactoryPullRequestMergeability) {
  return {
    msw: {
      handlers: [
        http.get("*/account/experimental-features", () =>
          HttpResponse.json({ features: [{ id: "factory_pull_request_merge", released: true }] }),
        ),
        http.get("*/api/v1/organizations/org-1", () =>
          HttpResponse.json({ organization: { metadata: { id: "org-1" } } }),
        ),
        http.get("*/api/v1/factories/factory-1/prs/pr-6812/mergeability", () => HttpResponse.json({ mergeability })),
      ],
    },
  };
}

export const MergeUnavailable: Story = {
  name: "Merge status is unavailable",
  parameters: statusParameters({
    canMerge: false,
    blockedReason: "BLOCKED_REASON_UNAVAILABLE",
    message: "Merge status is unavailable right now.",
  }),
  render: () => (
    <div className="max-w-[750px] overflow-hidden rounded-lg border border-border bg-background">
      <SplitRunAttentionNote
        note={{
          headline: "Waiting for user review",
          text: "The pull request is open.",
          cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
        }}
        organizationId="org-1"
        factoryId="factory-1"
        pullRequests={[
          {
            id: "pr-6812",
            provider: "PROVIDER_GITHUB",
            state: "STATE_OPEN",
            url: "https://github.com/acme/payments/pull/6812",
            number: "6812",
          },
        ]}
      />
    </div>
  ),
};

export const StatusReady: Story = {
  ...MergeUnavailable,
  name: "Status enabled — ready for provider review",
  parameters: statusParameters({ canMerge: true }),
};

export const ChecksRunning: Story = {
  ...MergeUnavailable,
  name: "Status enabled — checks running",
  parameters: statusParameters({
    canMerge: false,
    blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
    message: "Checks are still running.",
  }),
};

export const Conflicts: Story = {
  ...MergeUnavailable,
  name: "Status enabled — conflicts",
  parameters: statusParameters({
    canMerge: false,
    blockedReason: "BLOCKED_REASON_CONFLICTING",
    message: "The pull request has conflicts.",
  }),
};
