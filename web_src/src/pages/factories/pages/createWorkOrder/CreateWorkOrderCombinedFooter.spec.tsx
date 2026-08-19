import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { CreateWorkOrderCombinedFooter } from "./CreateWorkOrderCombinedFooter";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [], isLoading: false }),
}));

function renderFooter(overrides: Partial<Parameters<typeof CreateWorkOrderCombinedFooter>[0]> = {}) {
  const onSendToLine = vi.fn();
  const onSaveDraft = vi.fn();
  render(
    <CreateWorkOrderCombinedFooter
      organizationId="org-1"
      assigneeIds={[]}
      lines={REFUND_FACTORY_LINES}
      isSaving={false}
      canDispatch
      canSaveDraft
      isSavingDraft={false}
      isSendingToLine={false}
      onAssigneeChange={vi.fn()}
      onSaveDraft={onSaveDraft}
      onSendToLine={onSendToLine}
      {...overrides}
    />,
  );
  return { onSendToLine, onSaveDraft };
}

describe("CreateWorkOrderCombinedFooter", () => {
  it("puts Save as draft next to Send to line, no separate Line control", () => {
    renderFooter();

    expect(screen.getByTestId("work-order-create-draft-button")).toHaveTextContent("Save as draft");
    expect(screen.getByTestId("work-order-create-send-to-line")).toHaveTextContent("Send to line");
    expect(screen.queryByTestId("work-order-line-button")).not.toBeInTheDocument();
  });

  it("opens the line list from Send to line and sends on pick", async () => {
    const user = userEvent.setup();
    const { onSendToLine } = renderFooter();

    await user.click(screen.getByTestId("work-order-create-send-to-line"));

    const option = screen.getByTestId("work-order-line-option-plan-and-implement");
    expect(option).toBeInTheDocument();

    await user.click(option);

    expect(onSendToLine).toHaveBeenCalledWith("plan-and-implement");
  });

  it("disables Send to line and hints when the workspace has no lines", () => {
    renderFooter({ lines: [] });

    const button = screen.getByTestId("work-order-create-send-to-line");
    expect(button).toBeDisabled();
  });

  it("keeps Save as draft enabled without a line while Send to line needs one", () => {
    renderFooter({ lines: [], canSaveDraft: true });

    expect(screen.getByTestId("work-order-create-draft-button")).toBeEnabled();
    expect(screen.getByTestId("work-order-create-send-to-line")).toBeDisabled();
  });
});
