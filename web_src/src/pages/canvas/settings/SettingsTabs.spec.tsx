import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { SettingsTabs } from "./SettingsTabs";

describe("SettingsTabs", () => {
  it("marks General active on the settings root path", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/apps/app-1/settings"]}>
        <SettingsTabs organizationId="org-1" appId="app-1" />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("canvas-settings-tab-general")).toHaveAttribute("href", "/org-1/apps/app-1/settings");
    expect(screen.getByTestId("canvas-settings-tab-secrets")).toHaveAttribute(
      "href",
      "/org-1/apps/app-1/settings/secrets",
    );
  });

  it("marks Secrets active on the secrets path", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/apps/app-1/settings/secrets"]}>
        <SettingsTabs organizationId="org-1" appId="app-1" />
      </MemoryRouter>,
    );

    const secretsTab = screen.getByTestId("canvas-settings-tab-secrets");
    expect(secretsTab.className).toContain("border-sky-500");
  });

  it("marks Secrets active on a secret detail path", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/apps/app-1/settings/secrets/secret-1"]}>
        <SettingsTabs organizationId="org-1" appId="app-1" />
      </MemoryRouter>,
    );

    const secretsTab = screen.getByTestId("canvas-settings-tab-secrets");
    expect(secretsTab.className).toContain("border-sky-500");
  });
});
