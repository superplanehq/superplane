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

  it("commits staged edits before publishing so the agent's staged changes are included", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, text: async () => "" } as Response);
    vi.stubGlobal("fetch", fetchMock);
    const onDismiss = vi.fn();

    render(
      <DraftActionsWidget
        versionId="draft-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
    );

    await user.click(screen.getByRole("button", { name: /publish/i }));

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/canvases/canvas-1/versions/draft-1/staging/commit");
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: "POST" });
    expect(fetchMock.mock.calls[1][0]).toBe("/api/v1/canvases/canvas-1/versions/draft-1/publish");
    expect(fetchMock.mock.calls[1][1]).toMatchObject({ method: "PATCH" });
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("does not publish when committing staged edits fails", async () => {
    const user = userEvent.setup();
    vi.spyOn(console, "error").mockImplementation(() => {});
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 400, text: async () => "bad staging" } as Response);
    vi.stubGlobal("fetch", fetchMock);
    const onDismiss = vi.fn();

    render(
      <DraftActionsWidget
        versionId="draft-1"
        canvasId="canvas-1"
        organizationId="org-1"
        isEditing={false}
        onDismiss={onDismiss}
      />,
    );

    await user.click(screen.getByRole("button", { name: /publish/i }));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/canvases/canvas-1/versions/draft-1/staging/commit");
    expect(onDismiss).not.toHaveBeenCalled();
  });
});
