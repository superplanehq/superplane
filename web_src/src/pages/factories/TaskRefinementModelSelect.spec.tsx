import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, expect, it, vi } from "bun:test";

import { FactoryTaskScope } from "./FactoryTaskScope";
import { TaskRefinementModelSelect } from "./TaskRefinementModelSelect";
import { PlanningFailureAlert } from "./pages/work-order-split-run/PlanningFailureAlert";

const updateModel = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrder: () => ({ data: { state: "STATE_DRAFT", refinementModel: "gpt-5.6-luna" } }),
  useUpdateWorkOrder: () => ({ mutate: updateModel, isPending: false }),
}));

vi.mock("./useWorkOrderRefinementModels", () => ({
  useWorkOrderRefinementModels: () => ({
    defaultModel: "gpt-5.6-luna",
    options: [
      { value: "gpt-5.6-luna", label: "gpt-5.6-luna" },
      { value: "gpt-6.1-sol", label: "gpt-6.1-sol" },
    ],
  }),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

it("does not save the selected planning model again", async () => {
  updateModel.mockReset();
  render(
    <FactoryTaskScope factoryId="factory-1" orderId="order-1">
      <TaskRefinementModelSelect organizationId="org-1" canEdit />
    </FactoryTaskScope>,
  );

  await userEvent.click(screen.getByRole("button", { name: "Model: gpt-5.6-luna" }));
  await userEvent.click(screen.getByRole("menuitemradio", { name: "gpt-5.6-luna" }));
  expect(updateModel).not.toHaveBeenCalled();

  await userEvent.click(screen.getByRole("button", { name: "Model: gpt-5.6-luna" }));
  await userEvent.click(screen.getByRole("menuitemradio", { name: "gpt-6.1-sol" }));
  expect(updateModel).toHaveBeenCalledWith({ orderId: "order-1", refinementModel: "gpt-6.1-sol" }, expect.anything());
});

it("lets a failed task retry its current model from the alert and shows a save error", async () => {
  updateModel
    .mockReset()
    .mockImplementation((_input, options) => options.onError(new Error("Could not save this model.")));
  render(
    <FactoryTaskScope factoryId="factory-1" orderId="order-1">
      <PlanningFailureAlert
        failure={
          'model "gpt-5.6-luna": the selected model is unavailable for this task owner; select another model or clear the override'
        }
        organizationId="org-1"
        canEditModel
      />
    </FactoryTaskScope>,
  );

  const alert = screen.getByRole("alert", { name: "Planning failed" });
  expect(alert).toHaveTextContent('Model "gpt-5.6-luna" is not available for the current task owner.');
  expect(screen.queryByRole("button", { name: "Try planning again" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Model: gpt-5.6-luna" }));
  await userEvent.click(screen.getByRole("menuitemradio", { name: "gpt-5.6-luna" }));
  expect(updateModel).toHaveBeenCalledWith({ orderId: "order-1", refinementModel: "gpt-5.6-luna" }, expect.anything());
  expect(alert).toHaveTextContent("Could not save this model.");
});
