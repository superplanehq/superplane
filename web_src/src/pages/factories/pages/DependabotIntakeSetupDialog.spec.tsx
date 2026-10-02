import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DependabotIntakeSetupDialog } from "./DependabotIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  importItem: vi.fn(),
  searchSetupItems: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: mocks.importItem, isPending: false }),
  useSearchDependabotIntakeSetupItems: (input: unknown) => mocks.searchSetupItems(input),
}));

const ASTRO = {
  id: "npm:astro",
  key: "3 alerts",
  title: "Fix Dependabot alerts for astro (npm)",
  body: "",
  url: "https://github.com/acme/payments/security/dependabot?q=is%3Aopen+package%3Aastro+ecosystem%3Anpm",
};
const FFLATE = {
  id: "npm:fflate",
  key: "1 alert",
  title: "Fix Dependabot alerts for fflate (npm)",
  body: "",
  url: "https://github.com/acme/payments/security/dependabot?q=is%3Aopen+package%3Afflate+ecosystem%3Anpm",
};

function renderDialog(overrides: Partial<Parameters<typeof DependabotIntakeSetupDialog>[0]> = {}) {
  const onCreated = vi.fn();
  render(
    <DependabotIntakeSetupDialog
      organizationId="org-1"
      factoryId="factory-1"
      repository="acme/payments"
      setupReady
      onClose={vi.fn()}
      onCreated={onCreated}
      {...overrides}
    />,
  );
  return { onCreated };
}

async function openPicker(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { name: "Which open alerts should become tasks?" });
}

describe("DependabotIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.importItem.mockReset();
    mocks.importItem.mockResolvedValue({ id: "order-1" });
    mocks.searchSetupItems.mockReset();
    mocks.searchSetupItems.mockReturnValue({
      data: [ASTRO, FFLATE],
      isLoading: false,
      isError: false,
      refetch: mocks.refetch,
    });
  });

  it("opens the package picker without creating an intake", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();

    expect(screen.getByTestId("dependabot-setup-repository")).toHaveTextContent("acme/payments");
    await user.click(screen.getByRole("checkbox", { name: "Low" }));
    await openPicker(user);

    expect(mocks.createIntake).not.toHaveBeenCalled();
    expect(mocks.searchSetupItems).toHaveBeenCalled();
    expect(screen.getByTestId("dependabot-import-count")).toHaveTextContent(
      "2 packages have open alerts that match your filters.",
    );
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("creates the intake and imports only the selected packages when finished", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await openPicker(user);

    await user.click(screen.getByRole("checkbox", { name: /astro/ }));
    await user.click(screen.getByTestId("dependabot-import-selected"));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(mocks.createIntake).toHaveBeenCalledWith({
      source: "SOURCE_DEPENDABOT_ALERTS",
      settings: { dependabotSeverities: [] },
      skipInitialImport: true,
    });
    expect(mocks.importItem).toHaveBeenCalledTimes(1);
    expect(mocks.importItem).toHaveBeenCalledWith({ intakeId: "intake-1", itemId: "npm:astro" });
  });

  it("offers a primary finish action when no packages match", async () => {
    mocks.searchSetupItems.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
      refetch: mocks.refetch,
    });
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await openPicker(user);

    expect(screen.getByTestId("dependabot-import-empty")).toHaveTextContent("SuperPlane can still listen");
    expect(screen.queryByTestId("dependabot-import-selected")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Turn on Dependabot intake" }));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(mocks.createIntake).toHaveBeenCalledTimes(1);
    expect(mocks.importItem).not.toHaveBeenCalled();
  });

  it("creates the intake when the user skips the import", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await openPicker(user);

    await user.click(screen.getByTestId("dependabot-import-skip"));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(mocks.createIntake).toHaveBeenCalledTimes(1);
    expect(mocks.importItem).not.toHaveBeenCalled();
  });

  it("keeps the picker open when creation fails on finish", async () => {
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "permission denied" } } });
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await openPicker(user);

    await user.click(screen.getByTestId("dependabot-import-skip"));

    expect(await screen.findByRole("alert")).toHaveTextContent("permission denied");
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("shows why GitHub refused the open alerts", async () => {
    mocks.searchSetupItems.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: {
        response: {
          data: {
            message: "Dependabot alerts are off for this repository. Turn them on in the repository security settings.",
          },
        },
      },
      refetch: mocks.refetch,
    });
    const user = userEvent.setup();
    renderDialog();
    await openPicker(user);

    expect(screen.getByRole("alert")).toHaveTextContent("Turn them on in the repository security settings.");
    expect(screen.getByTestId("dependabot-open-security-settings")).toHaveAttribute(
      "href",
      "https://github.com/acme/payments/settings/security_analysis",
    );
  });

  it("blocks step one until workspace GitHub setup is complete", () => {
    renderDialog({ repository: "", setupReady: false });

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Connect GitHub and select a backlog repository in workspace setup before you create this intake.",
    );
    expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();
    expect(mocks.createIntake).not.toHaveBeenCalled();
  });
});
