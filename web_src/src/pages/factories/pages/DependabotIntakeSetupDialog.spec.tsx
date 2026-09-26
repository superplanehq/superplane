import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DependabotIntakeSetupDialog } from "./DependabotIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  importItem: vi.fn(),
  searchItems: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: mocks.importItem, isPending: false }),
  useSearchFactoryIntakeItems: (input: unknown) => mocks.searchItems(input),
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

async function createIntakeAndOpenPicker(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "Create intake" }));
  await screen.findByRole("heading", { name: "Which open alerts should become tasks?" });
}

describe("DependabotIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.importItem.mockReset();
    mocks.importItem.mockResolvedValue({ id: "order-1" });
    mocks.searchItems.mockReset();
    mocks.searchItems.mockReturnValue({
      data: [ASTRO, FFLATE],
      isLoading: false,
      isError: false,
      refetch: mocks.refetch,
    });
  });

  it("creates the intake without an initial import and opens the package picker", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();

    expect(screen.getByTestId("dependabot-setup-repository")).toHaveTextContent("acme/payments");
    expect(screen.getByTestId("dependabot-setup-instructions-note")).toHaveTextContent(
      "You can change this text in the intake settings.",
    );
    expect(screen.queryByTestId("dependabot-skip-initial-import")).not.toBeInTheDocument();
    await user.click(screen.getByRole("checkbox", { name: "Low" }));
    await createIntakeAndOpenPicker(user);

    expect(mocks.createIntake).toHaveBeenCalledWith({
      source: "SOURCE_DEPENDABOT_ALERTS",
      settings: { dependabotSeverities: ["critical", "high", "medium"] },
      skipInitialImport: true,
    });
    expect(mocks.searchItems).toHaveBeenLastCalledWith(
      expect.objectContaining({ intakeId: "intake-1", query: "", limit: 50 }),
    );
    expect(screen.getByTestId("dependabot-import-count")).toHaveTextContent(
      "2 packages have open alerts that match your filters.",
    );
    const list = screen.getByTestId("dependabot-import-packages");
    expect(within(list).getByText("Fix Dependabot alerts for astro (npm)")).toBeInTheDocument();
    expect(within(list).getByText("3 alerts")).toBeInTheDocument();
    expect(screen.getByTestId("dependabot-import-selected")).toHaveTextContent("Import selected (0)");
    expect(screen.getByTestId("dependabot-import-selected")).toBeDisabled();
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("imports only the selected packages and then leaves setup", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await createIntakeAndOpenPicker(user);

    await user.click(screen.getByRole("checkbox", { name: /astro/ }));
    expect(screen.getByTestId("dependabot-import-selected")).toHaveTextContent("Import selected (1)");
    await user.click(screen.getByTestId("dependabot-import-selected"));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(mocks.importItem).toHaveBeenCalledTimes(1);
    expect(mocks.importItem).toHaveBeenCalledWith({ intakeId: "intake-1", itemId: "npm:astro" });
  });

  it("selects and clears every package with one control", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await createIntakeAndOpenPicker(user);

    await user.click(screen.getByTestId("dependabot-import-select-all"));
    expect(screen.getByTestId("dependabot-import-selected")).toHaveTextContent("Import selected (2)");
    expect(screen.getByTestId("dependabot-import-select-all")).toHaveTextContent("Clear selection");

    await user.click(screen.getByTestId("dependabot-import-select-all"));
    expect(screen.getByTestId("dependabot-import-selected")).toHaveTextContent("Import selected (0)");

    await user.click(screen.getByTestId("dependabot-import-select-all"));
    await user.click(screen.getByTestId("dependabot-import-selected"));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(mocks.importItem.mock.calls.map((call) => call[0])).toEqual([
      { intakeId: "intake-1", itemId: "npm:astro" },
      { intakeId: "intake-1", itemId: "npm:fflate" },
    ]);
  });

  it("skips the import and leaves setup without creating tasks", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await createIntakeAndOpenPicker(user);

    await user.click(screen.getByTestId("dependabot-import-skip"));

    expect(onCreated).toHaveBeenCalledTimes(1);
    expect(mocks.importItem).not.toHaveBeenCalled();
  });

  it("keeps the picker open and reports the failure when an import fails", async () => {
    mocks.importItem.mockRejectedValue({ response: { data: { message: "rate limited" } } });
    const user = userEvent.setup();
    const { onCreated } = renderDialog();
    await createIntakeAndOpenPicker(user);

    await user.click(screen.getByRole("checkbox", { name: /fflate/ }));
    await user.click(screen.getByTestId("dependabot-import-selected"));

    expect(await screen.findByRole("alert")).toHaveTextContent("rate limited");
    expect(onCreated).not.toHaveBeenCalled();
    expect(screen.getByTestId("dependabot-import-selected")).toHaveTextContent("Import selected (1)");
  });

  it("shows an empty state when no open alerts match the filters", async () => {
    mocks.searchItems.mockReturnValue({ data: [], isLoading: false, isError: false, refetch: mocks.refetch });
    const user = userEvent.setup();
    renderDialog();
    await createIntakeAndOpenPicker(user);

    expect(screen.getByTestId("dependabot-import-empty")).toHaveTextContent("No open alerts match your filters.");
    expect(screen.queryByTestId("dependabot-import-packages")).not.toBeInTheDocument();
    expect(screen.getByTestId("dependabot-import-selected")).toBeDisabled();
    expect(screen.getByTestId("dependabot-import-skip")).toBeEnabled();
  });

  it("offers a retry when the open alerts do not load", async () => {
    mocks.searchItems.mockReturnValue({ data: undefined, isLoading: false, isError: true, refetch: mocks.refetch });
    const user = userEvent.setup();
    renderDialog();
    await createIntakeAndOpenPicker(user);

    expect(screen.getByRole("alert")).toHaveTextContent("SuperPlane could not load the open alerts.");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(mocks.refetch).toHaveBeenCalledTimes(1);
  });

  it("keeps the setup page open when creation fails", async () => {
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "permission denied" } } });
    const user = userEvent.setup();
    const { onCreated } = renderDialog();

    await user.click(screen.getByRole("button", { name: "Create intake" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("permission denied");
    expect(onCreated).not.toHaveBeenCalled();
    expect(mocks.searchItems).not.toHaveBeenCalledWith(expect.objectContaining({ intakeId: "intake-1" }));
  });

  it("blocks creation until workspace GitHub setup is complete", () => {
    renderDialog({ repository: "", setupReady: false });

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Connect GitHub and select a backlog repository in workspace setup before you create this intake.",
    );
    expect(screen.getByRole("button", { name: "Create intake" })).toBeDisabled();
    expect(mocks.createIntake).not.toHaveBeenCalled();
  });
});
