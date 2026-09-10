import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeAll, describe, expect, it } from "vitest";

import { TooltipProvider } from "@/components/ui/tooltip";

import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import type { SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";

const PR_NOTE: SplitRunFooterNote = {
  headline: "Waiting for user review",
  text: "The pull request is open and waiting for user review. Mention @superplaneagent in a pull request comment or review to request changes.",
  cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
};

const ACTIONS: SplitRunFooterAction[] = [
  { id: "back-to-draft", kind: "back-to-draft", label: "To Backlog", emphasis: "quiet", icon: "undo-2" },
  { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" },
  { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" },
];

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

function renderNote(props: Partial<Parameters<typeof SplitRunAttentionNote>[0]> = {}) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <SplitRunAttentionNote note={PR_NOTE} tone="waiting" actions={ACTIONS} {...props} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("SplitRunAttentionNote for a pull request", () => {
  it("says the pull request is ready and lists the three review steps", () => {
    renderNote();

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();

    const steps = within(within(note).getByRole("list", { name: "Next steps" })).getAllByRole("listitem");
    expect(steps).toHaveLength(3);
    expect(steps[0]).toHaveTextContent("1");
    expect(steps[0]).toHaveTextContent("Review the pull request");
    expect(steps[1]).toHaveTextContent("2");
    expect(steps[1]).toHaveTextContent("Leave comments");
    expect(steps[1]).toHaveTextContent("@superplaneagent");
    expect(steps[2]).toHaveTextContent("3");
    expect(steps[2]).toHaveTextContent("SuperPlane addresses them");

    expect(note).not.toHaveTextContent("Waiting for user review");
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
  });

  it("makes the pull request link the one large call to action", () => {
    renderNote();

    const cta = screen.getByTestId("split-run-pull-request-cta");
    expect(cta).toHaveAccessibleName("Review PR #6812");
    expect(cta).toHaveAttribute("href", "https://github.com/acme/payments/pull/6812");
    expect(cta).toHaveAttribute("target", "_blank");
    expect(cta).toHaveAttribute("rel", "noreferrer");

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getAllByRole("link")).toHaveLength(1);
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "To Backlog" })).not.toBeInTheDocument();
  });

  it("has no More menu, even when close actions are available", () => {
    renderNote();

    expect(screen.queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-pull-request-cta")).toBeInTheDocument();
  });

  it("keeps the standard note for a link that is not a pull request", () => {
    renderNote({
      note: { headline: "Implement did not pass", text: "Fix the error.", cta: { label: "Debug", href: "/runs/1" } },
      tone: "failed",
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(note).not.toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "Implement did not pass" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });
});
