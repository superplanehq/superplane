import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { mutateAsync, useWorkOrderArtifacts } = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
  useWorkOrderArtifacts: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useSendWorkOrderToBacklog: () => ({ mutateAsync, isPending: false }),
  useWorkOrderArtifacts: (...args: unknown[]) => useWorkOrderArtifacts(...args),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { SendWorkOrderToBacklogForm } from "./SendWorkOrderToBacklogForm";

describe("SendWorkOrderToBacklogForm", () => {
  beforeEach(() => {
    mutateAsync.mockReset().mockResolvedValue({ id: "wo-1", state: "STATE_DRAFT" });
    useWorkOrderArtifacts.mockReset().mockReturnValue({ data: [], isLoading: false });
    vi.mocked(showSuccessToast).mockReset();
    vi.mocked(showErrorToast).mockReset();
  });

  it("hides both checkboxes when the task has no closeable pull requests or artifacts", async () => {
    const user = userEvent.setup();
    render(
      <SendWorkOrderToBacklogForm
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
        pullRequests={[{ id: "pr-1", state: "STATE_MERGED" }]}
      />,
    );

    expect(screen.queryByTestId("send-to-backlog-close-prs")).not.toBeInTheDocument();
    expect(screen.queryByTestId("send-to-backlog-clear-artifacts")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("send-to-backlog-submit"));
    expect(mutateAsync).toHaveBeenCalledWith({
      orderId: "wo-1",
      closePullRequests: false,
      clearArtifacts: false,
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Task sent to Backlog.");
  });

  it("shows unchecked checkboxes and sends the chosen flags", async () => {
    const user = userEvent.setup();
    useWorkOrderArtifacts.mockReturnValue({
      data: [{ id: "art-1", type: "TYPE_MARKDOWN" }],
      isLoading: false,
    });

    render(
      <SendWorkOrderToBacklogForm
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
        pullRequests={[{ id: "pr-1", state: "STATE_OPEN" }]}
      />,
    );

    const closePrs = screen.getByTestId("send-to-backlog-close-prs");
    const clearArtifacts = screen.getByTestId("send-to-backlog-clear-artifacts");
    expect(closePrs).not.toBeChecked();
    expect(clearArtifacts).not.toBeChecked();

    await user.click(closePrs);
    await user.click(screen.getByTestId("send-to-backlog-submit"));
    expect(mutateAsync).toHaveBeenCalledWith({
      orderId: "wo-1",
      closePullRequests: true,
      clearArtifacts: false,
    });
  });
});
