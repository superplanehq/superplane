import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FleetManagerProviderSection, type FleetManagerSetupState } from "./FleetManagerSections";

function buildSetup({
  provider = "aws",
  terraformProvider = "aws",
}: { provider?: string; terraformProvider?: string } = {}) {
  const setProvider = vi.fn();
  const setup = {
    metadata: {
      infrastructureDefaults: { provider: terraformProvider },
    },
    draft: { provider },
    setProvider,
  } as unknown as FleetManagerSetupState;
  return { setup, setProvider };
}

describe("FleetManagerProviderSection", () => {
  it.each([
    { id: "aws", name: "AWS" },
    { id: "gcp", name: "GCP" },
    { id: "azure", name: "Azure" },
  ])("shows the $id logo on a button named $name", ({ id, name }) => {
    const { setup } = buildSetup();
    render(<FleetManagerProviderSection setup={setup} />);

    const card = screen.getByRole("button", { name });
    const logo = within(card).getByTestId(`provider-logo-${id}`);
    expect(logo).toHaveAttribute("aria-hidden", "true");
  });

  it("shows the provider name in a tooltip", async () => {
    const user = userEvent.setup();
    const { setup } = buildSetup();
    const { container } = render(<FleetManagerProviderSection setup={setup} />);

    expect(screen.queryByText("Azure")).not.toBeInTheDocument();
    await user.hover(screen.getByRole("button", { name: "Azure" }));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Azure");
    expect(container.querySelector(".lucide-cloud")).not.toBeInTheDocument();
  });

  it("announces the Terraform note and shows it in a tooltip", async () => {
    const user = userEvent.setup();
    const { setup } = buildSetup({ terraformProvider: "gcp" });
    render(<FleetManagerProviderSection setup={setup} />);

    const gcpCard = screen.getByRole("button", { name: "GCP", description: "Loaded from Terraform" });
    expect(screen.getByRole("button", { name: "AWS" })).not.toHaveAccessibleDescription();
    expect(screen.getByRole("button", { name: "Azure" })).not.toHaveAccessibleDescription();

    await user.hover(gcpCard);
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip).toHaveTextContent("GCP");
    expect(tooltip).not.toHaveTextContent("Loaded from Terraform");
    expect(screen.getAllByRole("tooltip")).toHaveLength(1);
  });

  it("selects the provider on click", async () => {
    const user = userEvent.setup();
    const { setup, setProvider } = buildSetup();
    render(<FleetManagerProviderSection setup={setup} />);

    await user.click(screen.getByRole("button", { name: "This VM" }));
    expect(setProvider).toHaveBeenCalledWith("docker");
  });
});
