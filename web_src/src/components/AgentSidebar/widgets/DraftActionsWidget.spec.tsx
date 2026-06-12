import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DraftActionsWidget } from "./DraftActionsWidget";

describe("DraftActionsWidget", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("dispatches a view event when the user asks to see changes", async () => {
    const user = userEvent.setup();
    const dispatchEvent = vi.spyOn(window, "dispatchEvent");

    render(<DraftActionsWidget versionId="draft-1" canvasId="canvas-1" organizationId="org-1" isEditing={false} />);

    await user.click(screen.getByRole("button", { name: /see changes/i }));

    expect(dispatchEvent).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "agent:view-version",
        detail: { versionId: "draft-1" },
      }),
    );
  });
});
