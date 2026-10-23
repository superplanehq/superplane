import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { WorkOrderExecutionsList } from "./WorkOrderExecutionsList";

describe("queued line dispatches", () => {
  it.each(["default", "compact", "inline"] as const)("shows a queued step without a run in the %s view", (variant) => {
    render(
      <WorkOrderExecutionsList
        organizationId="organization"
        factoryKey="workspace"
        variant={variant}
        dispatches={[
          {
            id: "dispatch",
            line: { id: "line", name: "Delivery" },
            queueItem: { id: "queued-step", stepName: "Build", position: 2 },
          },
        ]}
      />,
    );

    expect(screen.getByText(/Build/)).toBeVisible();
    expect(screen.getByText("Queued · position 2")).toBeVisible();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.queryByText(/No line runs yet/)).not.toBeInTheDocument();
  });
});
