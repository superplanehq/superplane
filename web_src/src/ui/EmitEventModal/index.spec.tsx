import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { EmitEventModal } from "@/ui/EmitEventModal";

const showErrorToast = vi.fn();
const showSuccessToast = vi.fn();

vi.mock("@/lib/toast", () => ({
  showErrorToast: (...args: unknown[]) => showErrorToast(...args),
  showSuccessToast: (...args: unknown[]) => showSuccessToast(...args),
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (value?: string) => void }) => (
    <textarea
      aria-label="event data"
      defaultValue={value}
      onChange={(event) => onChange?.(event.currentTarget.value)}
    />
  ),
}));

describe("EmitEventModal", () => {
  beforeEach(() => {
    showErrorToast.mockReset();
    showSuccessToast.mockReset();
  });

  it("shows a friendly fallback toast for network failures", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const onEmit = vi.fn().mockRejectedValue(new Error("Failed to fetch"));

    render(
      <EmitEventModal
        isOpen={true}
        onClose={vi.fn()}
        nodeId="node-1"
        nodeName="Node 1"
        workflowId="workflow-1"
        organizationId="org-1"
        channels={["default"]}
        onEmit={onEmit}
      />,
    );

    fireEvent.click(screen.getByTestId("emit-event-submit-button"));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Failed to emit event");
    });

    expect(consoleErrorSpy).not.toHaveBeenCalled();
    consoleErrorSpy.mockRestore();
  });
});
