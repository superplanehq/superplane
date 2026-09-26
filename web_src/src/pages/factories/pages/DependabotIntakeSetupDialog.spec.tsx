import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DependabotIntakeSetupDialog } from "./DependabotIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

describe("DependabotIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
  });

  it("creates a configured Dependabot intake without opening the automation", async () => {
    const onCreated = vi.fn();
    const user = userEvent.setup();
    render(
      <DependabotIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/payments"
        setupReady
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    expect(screen.getByTestId("dependabot-setup-repository")).toHaveTextContent("acme/payments");
    expect(screen.getByTestId("dependabot-setup-instructions-note")).toHaveTextContent(
      "You can change this text in the intake settings.",
    );
    await user.click(screen.getByRole("checkbox", { name: "Low" }));
    await user.click(screen.getByTestId("dependabot-skip-initial-import"));
    await user.click(screen.getByRole("button", { name: "Create intake" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_DEPENDABOT_ALERTS",
        settings: { dependabotSeverities: ["critical", "high", "medium"] },
        skipInitialImport: true,
      });
    });
    expect(onCreated).toHaveBeenCalledTimes(1);
  });

  it("keeps the setup page open when creation fails", async () => {
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "permission denied" } } });
    const onCreated = vi.fn();
    const user = userEvent.setup();
    render(
      <DependabotIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/payments"
        setupReady
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Create intake" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("permission denied");
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("blocks creation until workspace GitHub setup is complete", () => {
    render(
      <DependabotIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository=""
        setupReady={false}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Connect GitHub and select a backlog repository in workspace setup before you create this intake.",
    );
    expect(screen.getByRole("button", { name: "Create intake" })).toBeDisabled();
    expect(mocks.createIntake).not.toHaveBeenCalled();
  });
});
