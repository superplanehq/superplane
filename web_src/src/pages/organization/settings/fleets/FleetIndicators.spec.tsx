import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "bun:test";
import { FleetCapacity, FleetStatus } from "./FleetIndicators";

afterEach(cleanup);

for (const enabled of [true, false]) {
  it(`explains an ${enabled ? "enabled" : "disabled"} fleet on keyboard focus`, async () => {
    render(<FleetStatus enabled={enabled} />);
    const label = `This fleet is ${enabled ? "enabled" : "disabled"}.`;
    fireEvent.focus(screen.getByRole("button", { name: label }));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(label);
  });
}

it("explains the three runner counts without showing task or terminated counts", async () => {
  render(
    <FleetCapacity
      capacity={{ pendingRunners: "4", idleRunners: "1", busyRunners: "2", runnableTasks: "3", terminatedRunners: "5" }}
    />,
  );
  expect(screen.getAllByRole("button")).toHaveLength(3);
  for (const [label, explanation] of [
    ["Pending runners: 4", "Pending runners: 4. Waiting to register."],
    ["Idle runners: 1", "Idle runners: 1. Ready to accept work."],
    ["Busy runners: 2", "Busy runners: 2. Executing work."],
  ]) {
    const count = screen.getByRole("button", { name: label });
    fireEvent.focus(count);
    expect(await screen.findByRole("tooltip")).toHaveTextContent(explanation);
    fireEvent.blur(count);
  }
  expect(screen.queryByText("Runnable tasks")).not.toBeInTheDocument();
  expect(screen.queryByText("Terminated runners")).not.toBeInTheDocument();
});
