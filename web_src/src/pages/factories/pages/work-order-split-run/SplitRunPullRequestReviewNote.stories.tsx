import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { SplitRunPullRequestMergeControls } from "./SplitRunPullRequestMergeAction";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const meta = {
  title: "Factories/Pages/Task Split Run/Pull request review strip",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-muted/30 p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

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

export const MergeUnavailable: Story = {
  name: "Merge status is unavailable",
  parameters: {
    msw: {
      handlers: [
        http.get("*/account/experimental-features", () =>
          HttpResponse.json({ features: [{ id: "factory_pull_request_merge", released: true }] }),
        ),
        http.get("*/api/v1/organizations/org-1", () =>
          HttpResponse.json({ organization: { metadata: { id: "org-1" } } }),
        ),
        http.get("*/api/v1/factories/factory-1/prs/pr-6812/mergeability", () =>
          HttpResponse.json({
            mergeability: {
              canMerge: false,
              blockedReason: "BLOCKED_REASON_UNAVAILABLE",
              message: "Merge status is unavailable right now.",
            },
          }),
        ),
      ],
    },
  },
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
        orderId="wo-1"
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

export const WebhookFailed: Story = {
  name: "Webhook setup failed",
  parameters: {
    msw: {
      handlers: [
        http.get("*/account/experimental-features", () =>
          HttpResponse.json({ features: [{ id: "factory_pull_request_merge", released: true }] }),
        ),
        http.get("*/api/v1/organizations/org-1", () =>
          HttpResponse.json({ organization: { metadata: { id: "org-1" } } }),
        ),
        http.get("*/api/v1/factories/factory-1/prs/pr-6812/mergeability", () =>
          HttpResponse.json({
            mergeability: {
              canMerge: false,
              blockedReason: "BLOCKED_REASON_WEBHOOK_FAILED",
              message:
                "SuperPlane could not register a webhook on this repository. GitHub allows 20 pull request webhooks, and this repository already has 20. Remove an unused webhook, then try again.",
            },
          }),
        ),
        http.post("*/api/v1/factories/factory-1/prs/pr-6812/webhook-retry", () => HttpResponse.json({})),
      ],
    },
  },
  render: () => (
    <div className="flex max-w-[750px] flex-col gap-6">
      <SplitRunAttentionNote
        note={{
          headline: "Waiting for user review",
          text: "The pull request is open.",
          cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
        }}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
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
      <SplitRunAttentionNote
        compact
        note={{
          headline: "Waiting for user review",
          text: "The pull request is open.",
          cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
        }}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
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

export const MergeReady: Story = {
  name: "Merge is ready",
  render: () => (
    <PopupShell>
      <div className="border-t border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] px-5 py-5">
        <h3 className="text-[18px] font-semibold">The pull request is ready for review</h3>
        <div className="mt-4 flex flex-wrap items-center gap-4">
          <SplitRunPullRequestMergeControls
            mergeability={{
              canMerge: true,
              allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_MERGE"],
              headSha: "abc123",
            }}
            merging={false}
            canAct
            compact={false}
            onMerge={() => {}}
          />
        </div>
      </div>
    </PopupShell>
  ),
};

export const MergeBlocked: Story = {
  name: "Merge is blocked",
  render: () => (
    <PopupShell>
      <div className="border-t border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] px-5 py-5">
        <h3 className="text-[18px] font-semibold">The pull request is ready for review</h3>
        <div className="mt-4 flex flex-wrap items-center gap-4">
          <SplitRunPullRequestMergeControls
            mergeability={{
              canMerge: false,
              blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
              message: "Checks are still running.",
              allowedMethods: ["MERGE_METHOD_SQUASH"],
              headSha: "abc123",
            }}
            merging={false}
            canAct
            compact={false}
            onMerge={() => {}}
          />
        </div>
      </div>
    </PopupShell>
  ),
};
