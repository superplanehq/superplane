import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { CreateWorkOrderRequestDialog } from "./CreateWorkOrderRequestDialog";

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: () => <textarea id="work-order-description-input" />,
}));

const modelSelection = {
  defaultModel: "sonnet",
  options: [
    { value: "sonnet", label: "Claude Sonnet" },
    { value: "opus", label: "Claude Opus" },
  ],
};

function renderDialog(selection = modelSelection, initialModel?: string) {
  const onCreate = vi.fn();
  const view = render(
    <CreateWorkOrderRequestDialog
      open
      description="Refunds fail."
      maxLength={5000}
      onClose={vi.fn()}
      onDescriptionChange={vi.fn()}
      onCreate={onCreate}
      modelSelection={selection}
      initialModel={initialModel}
    />,
  );
  return { ...view, onCreate };
}

describe("Request composer refinement model", () => {
  afterEach(cleanup);

  it("includes a different model and omits it when the phase default is selected again", async () => {
    const user = userEvent.setup();
    const { onCreate } = renderDialog();
    await user.click(screen.getByRole("button", { name: "Model: Claude Sonnet" }));
    await user.click(screen.getByRole("menuitemradio", { name: "Claude Opus" }));
    await user.click(screen.getByTestId("create-work-order-request-create"));
    expect(onCreate).toHaveBeenLastCalledWith({ title: "Refunds fail.", description: "Refunds fail.", model: "opus" });
    await user.click(screen.getByRole("button", { name: "Model: Claude Opus" }));
    await user.click(screen.getByRole("menuitemradio", { name: "Claude Sonnet" }));
    await user.click(screen.getByTestId("create-work-order-request-create"));
    expect(onCreate).toHaveBeenLastCalledWith({ title: "Refunds fail.", description: "Refunds fail." });
  });

  it.each([true, false])("does not send an override when models are unavailable (loading: %s)", async (isLoading) => {
    const onCreate = vi.fn();
    render(
      <CreateWorkOrderRequestDialog
        open
        description="Refunds fail."
        maxLength={5000}
        onClose={vi.fn()}
        onDescriptionChange={vi.fn()}
        onCreate={onCreate}
        initialModel="opus"
        modelSelection={{ defaultModel: "sonnet", options: isLoading ? modelSelection.options : [], isLoading }}
      />,
    );
    expect(screen.getByRole("button", { name: /^Model:/ })).toBeDisabled();
    await userEvent.setup().click(screen.getByTestId("create-work-order-request-create"));
    expect(onCreate).toHaveBeenCalledWith({ title: "Refunds fail.", description: "Refunds fail." });
  });
});
