import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvasVersion } from "@/api-client";

import { StartEditingDropdown } from "./StartEditingDropdown";

function draft(overrides: Partial<CanvasesCanvasVersion> = {}): CanvasesCanvasVersion {
  return {
    metadata: {
      branchName: "drafts/user-1",
      displayName: "My draft",
      owner: { id: "user-1", name: "Ada Lovelace" },
      updatedAt: "2026-06-03T12:00:00.000Z",
      ...overrides.metadata,
    },
    ...overrides,
  };
}

describe("StartEditingDropdown", () => {
  it("creates a draft directly from the Edit button when there are no drafts", async () => {
    const user = userEvent.setup();
    const onCreateDraft = vi.fn();

    render(
      <StartEditingDropdown
        open
        onOpenChange={vi.fn()}
        drafts={[]}
        defaultDraft={null}
        onContinueDraft={vi.fn()}
        onCreateDraft={onCreateDraft}
      />,
    );

    expect(screen.queryByTestId("start-editing-menu")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("canvas-edit-button"));
    expect(onCreateDraft).toHaveBeenCalledTimes(1);
  });

  it("shows continue and create when one draft exists", async () => {
    const user = userEvent.setup();
    const onContinueDraft = vi.fn();
    const existingDraft = draft();

    render(
      <StartEditingDropdown
        open
        onOpenChange={vi.fn()}
        drafts={[existingDraft]}
        defaultDraft={existingDraft}
        onContinueDraft={onContinueDraft}
        onCreateDraft={vi.fn()}
      />,
    );

    expect(screen.getByTestId("start-editing-continue")).toHaveTextContent("Continue My draft");
    expect(screen.queryByTestId("start-editing-choose-list")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("start-editing-continue"));
    expect(onContinueDraft).toHaveBeenCalledWith("drafts/user-1");
  });

  it("shows choose-from-list when multiple drafts exist", async () => {
    const user = userEvent.setup();
    const onContinueDraft = vi.fn();
    const drafts = [
      draft({ metadata: { branchName: "drafts/user-1", displayName: "First draft" } }),
      draft({
        metadata: {
          branchName: "drafts/user-2",
          displayName: "Second draft",
          owner: { id: "user-2", name: "Grace Hopper" },
        },
      }),
    ];

    render(
      <StartEditingDropdown
        open
        onOpenChange={vi.fn()}
        drafts={drafts}
        defaultDraft={drafts[0]!}
        onContinueDraft={onContinueDraft}
        onCreateDraft={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("start-editing-choose-list"));

    expect(screen.getByText("Choose a draft")).toBeInTheDocument();
    expect(screen.getAllByTestId("start-editing-draft-row")).toHaveLength(2);

    await user.click(screen.getAllByTestId("start-editing-draft-row")[1]!);
    expect(onContinueDraft).toHaveBeenCalledWith("drafts/user-2");
  });
});
