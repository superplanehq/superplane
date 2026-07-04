import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DraftActionsWidget } from "./DraftActionsWidget";

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
