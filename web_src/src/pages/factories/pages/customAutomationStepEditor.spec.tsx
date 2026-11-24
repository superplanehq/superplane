import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { StepEditor } from "./customAutomationStepEditor";

vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => false }) }));
vi.mock("./PlanningReviewStepBody", () => ({ PlanningReviewStepBody: () => null }));

it("selects Chat for the final prompt step", async () => {
  const onChange = vi.fn();
  const view = render(
    <ThemeProvider>
      <StepEditor
        step={{ name: "Discuss", type: "prompt", prompt: "Discuss the task." }}
        position={0}
        stepCount={1}
        runsOnTask
        organizationId="org"
        onChange={onChange}
        onMove={() => {}}
        onRemove={() => {}}
      />
    </ThemeProvider>,
  );
  const user = userEvent.setup();
  await user.click(screen.getByRole("combobox", { name: "Type" }));
  await user.click(screen.getByRole("option", { name: "Chat" }));
  expect(onChange).toHaveBeenCalledWith(
    expect.objectContaining({ type: "prompt", continueInChat: true, prompt: "Discuss the task." }),
  );
  view.unmount();
});
