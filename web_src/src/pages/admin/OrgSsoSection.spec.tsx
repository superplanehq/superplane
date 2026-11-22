import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { OrgSsoSection } from "./OrgSsoSection";

const { getSso, disableSso } = vi.hoisted(() => ({
  getSso: vi.fn(),
  disableSso: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  adminSsoGetOrganizationSso: getSso,
  adminSsoDisableOrganizationSso: disableSso,
  adminSsoVerifyOrganizationDomain: vi.fn(),
}));

const connection = (status: string) => ({
  data: { connection: { exists: true, status, enforcement: "optional" }, domains: [] },
});

const renderSection = () =>
  render(
    <MemoryRouter>
      <OrgSsoSection orgId="org-1" />
    </MemoryRouter>,
  );

describe("OrgSsoSection", () => {
  beforeEach(() => {
    getSso.mockReset();
    disableSso.mockReset();
  });

  it("shows a load error", async () => {
    getSso.mockRejectedValue(new Error("unavailable"));
    renderSection();
    expect(await screen.findByRole("alert")).toHaveTextContent("Single Sign-On details are not available.");
  });

  it("shows the message when disable fails", async () => {
    getSso.mockResolvedValue(connection("active"));
    disableSso.mockRejectedValue(new Error("The connection could not be disabled."));
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Disable connection" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The connection could not be disabled.");
  });

  it("does not offer disable for a draft or disabled connection", async () => {
    getSso.mockResolvedValue(connection("draft"));
    const { unmount } = renderSection();
    expect(await screen.findByText("Status: draft")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disable connection" })).not.toBeInTheDocument();
    unmount();

    getSso.mockResolvedValue(connection("disabled"));
    renderSection();
    expect(await screen.findByText("Status: disabled")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disable connection" })).not.toBeInTheDocument();
  });
});
