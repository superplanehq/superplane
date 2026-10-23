import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { SplitRunReview } from "./SplitRunReview";
import { buildSplitRunFooter, type SplitRunFooterNote } from "./splitRunFooter";

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, enabledExperimentalFeatures: [], isLoading: false }),
}));

const PAUSED_NOTE = {
  key: "check-fixes-paused",
  headline: "Automatic fixes did not succeed",
  text: "SuperPlane paused automatic fixes. Continue fixes or review the pull request.",
};

const PULL_REQUEST_CTA: SplitRunFooterNote["cta"] = {
  label: "Review PR #6812",
  href: "https://github.com/acme/payments/pull/6812",
};

function pausedFooter(withPullRequest: boolean) {
  return buildSplitRunFooter({
    kind: "waiting",
    note: withPullRequest ? { ...PAUSED_NOTE, cta: PULL_REQUEST_CTA } : PAUSED_NOTE,
    continueFixes: { pullRequestId: "pr-1", runId: "run-paused" },
  });
}

function renderReview(props: Partial<Parameters<typeof SplitRunReview>[0]> & { withPullRequest?: boolean }) {
  const { withPullRequest = false, ...rest } = props;
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TooltipProvider>
          <SplitRunReview
            footer={pausedFooter(withPullRequest)}
            organizationId="org-1"
            factoryId="factory-1"
            {...rest}
          />
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe.each([
  ["the decision note", false],
  ["the pull request review strip", true],
])("Continue fixes on %s", (_name, withPullRequest) => {
  it("continues fixes without closing the task", async () => {
    const user = userEvent.setup();
    const onContinueFixes = vi.fn();
    const onReject = vi.fn();
    const onStop = vi.fn();
    renderReview({ withPullRequest, onContinueFixes, onReject, onStop });

    await user.click(screen.getByRole("button", { name: "Continue fixes" }));

    expect(onContinueFixes).toHaveBeenCalledTimes(1);
    expect(onReject).not.toHaveBeenCalled();
    expect(onStop).not.toHaveBeenCalled();
  });

  it("disables the button while another action runs", () => {
    renderReview({ withPullRequest, actionBusy: true });

    expect(screen.getByRole("button", { name: "Continue fixes" })).toBeDisabled();
  });

  it("hides the button from a person who cannot update the task", () => {
    renderReview({ withPullRequest, canAct: false });

    expect(screen.queryByRole("button", { name: "Continue fixes" })).not.toBeInTheDocument();
  });
});

describe("Continue fixes beside the close actions", () => {
  it("keeps Reject and Approve on the decision note", () => {
    renderReview({});

    expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual([
      "Continue fixes",
      "Reject",
      "Approve",
    ]);
  });
});
