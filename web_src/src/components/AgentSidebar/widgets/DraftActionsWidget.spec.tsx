import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DraftActionsWidget } from "./DraftActionsWidget";
import { draftActionsConfirmCopy } from "./draftActionsConfirmCopy";

describe("DraftActionsWidget", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls onViewStaging when the user asks to see changes", async () => {
    const user = userEvent.setup();
    const onViewStaging = vi.fn();

    render(
      <DraftActionsWidget canvasId="canvas-1" organizationId="org-1" isEditing={false} onViewStaging={onViewStaging} />,
    );

    await user.click(screen.getByRole("button", { name: /see changes/i }));

    expect(onViewStaging).toHaveBeenCalledTimes(1);
  });

  it("uses Save copy in factory context", () => {
    expect(draftActionsConfirmCopy("save")).toEqual({ idle: "Save", busy: "Saving..." });
    expect(draftActionsConfirmCopy("commit")).toEqual({ idle: "Commit", busy: "Committing..." });
  });

  it("shows Save instead of Commit when confirmKind is save", () => {
    render(
      <DraftActionsWidget
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing
        confirmKind="save"
        onCommitStaging={vi.fn().mockResolvedValue(true)}
      />,
    );

    expect(screen.getByRole("button", { name: /^save$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^commit$/i })).not.toBeInTheDocument();
  });

  it("commits staging through the shared commit handler", async () => {
    const user = userEvent.setup();
    const onCommitStaging = vi.fn().mockResolvedValue(true);
    const onDismiss = vi.fn();

    render(
      <DraftActionsWidget
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        message="Added health checks"
        onCommitStaging={onCommitStaging}
        onDismiss={onDismiss}
      />,
    );

    await user.click(screen.getByRole("button", { name: /commit/i }));

    expect(onCommitStaging).toHaveBeenCalledWith("Added health checks");
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("does not dismiss when committing staging fails", async () => {
    const user = userEvent.setup();
    const onCommitStaging = vi.fn().mockResolvedValue(false);
    const onDismiss = vi.fn();

    render(
      <DraftActionsWidget
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onCommitStaging={onCommitStaging}
        onDismiss={onDismiss}
      />,
    );

    await user.click(screen.getByRole("button", { name: /commit/i }));

    expect(onCommitStaging).toHaveBeenCalledTimes(1);
    expect(onDismiss).not.toHaveBeenCalled();
  });
});
