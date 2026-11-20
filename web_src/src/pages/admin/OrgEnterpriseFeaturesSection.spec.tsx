import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { OrgEnterpriseFeaturesSection } from "./OrgEnterpriseFeaturesSection";

const { listFeatures, setFeatures } = vi.hoisted(() => ({
  listFeatures: vi.fn(),
  setFeatures: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  enterpriseFeaturesListOrganizationEnterpriseFeatures: listFeatures,
  enterpriseFeaturesSetOrganizationEnterpriseFeatures: setFeatures,
}));

const groups = {
  key: "groups",
  licensed: true,
  granted: true,
  effective: true,
  source: "admin",
  grantedByName: "Ada Lovelace",
  createdAt: "2026-03-01T12:00:00Z",
};

const roles = {
  key: "custom_roles",
  licensed: true,
  granted: false,
  effective: false,
};

const renderSection = () =>
  render(
    <MemoryRouter>
      <OrgEnterpriseFeaturesSection orgId="org-1" />
    </MemoryRouter>,
  );

describe("OrgEnterpriseFeaturesSection", () => {
  beforeEach(() => {
    listFeatures.mockReset();
    setFeatures.mockReset();
  });

  it("renders licensed features and saves the full set", async () => {
    listFeatures.mockResolvedValue({ data: { licenseActive: true, features: [groups, roles] } });
    setFeatures.mockResolvedValue({
      data: { features: [{ ...roles, granted: true, effective: true }, groups] },
    });
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("Groups")).toBeInTheDocument();
    expect(screen.getByText("Custom roles")).toBeInTheDocument();
    expect(screen.getByText(/Enabled by Ada Lovelace on /)).toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: "Toggle Custom roles" }));

    expect(setFeatures).toHaveBeenCalledWith({
      path: { organizationId: "org-1" },
      body: { features: ["groups", "custom_roles"] },
    });
  });

  it("disables switches when there is no active license and still allows a grant to turn off", async () => {
    listFeatures.mockResolvedValue({
      data: {
        licenseActive: false,
        features: [
          { ...groups, licensed: false, effective: false },
          { ...roles, licensed: false },
        ],
      },
    });
    setFeatures.mockResolvedValue({ data: { features: [{ ...roles, licensed: false }] } });
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("No active Enterprise license.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "License" })).toHaveAttribute("href", "/admin/license");
    expect(screen.getByText("Not in the current license.")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Toggle Custom roles" })).toBeDisabled();

    await user.click(screen.getByRole("switch", { name: "Toggle Groups" }));

    expect(setFeatures).toHaveBeenCalledWith({
      path: { organizationId: "org-1" },
      body: { features: [] },
    });
  });

  it("shows the API error", async () => {
    listFeatures.mockResolvedValue({ data: { licenseActive: true, features: [groups, roles] } });
    setFeatures.mockRejectedValue({ message: "The license does not include custom_roles." });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("switch", { name: "Toggle Custom roles" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The license does not include custom_roles.");
  });
});
