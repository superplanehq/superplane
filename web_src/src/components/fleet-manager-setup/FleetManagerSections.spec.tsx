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

  it("does not show the provider names or the generic cloud icon", () => {
    const { setup } = buildSetup();
    const { container } = render(<FleetManagerProviderSection setup={setup} />);

    for (const name of ["AWS", "GCP", "AZURE"]) {
      expect(screen.queryByText(name)).not.toBeInTheDocument();
    }
    expect(container.querySelector(".lucide-cloud")).not.toBeInTheDocument();
  });

  it("keeps the Terraform note on the Terraform provider card", () => {
    const { setup } = buildSetup({ terraformProvider: "gcp" });
    render(<FleetManagerProviderSection setup={setup} />);

    const gcpCard = screen.getByRole("button", { name: "GCP" });
    expect(within(gcpCard).getByText("Loaded from Terraform")).toBeInTheDocument();
    expect(screen.getAllByText("Loaded from Terraform")).toHaveLength(1);
  });

  it("selects the provider on click", async () => {
    const user = userEvent.setup();
    const { setup, setProvider } = buildSetup();
    render(<FleetManagerProviderSection setup={setup} />);

    await user.click(screen.getByRole("button", { name: "Azure" }));
    expect(setProvider).toHaveBeenCalledWith("azure");
  });
});
