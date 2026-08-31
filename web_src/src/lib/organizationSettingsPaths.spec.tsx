import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  OrganizationSettingsPathsProvider,
  useOrganizationSettingsPaths,
  type OrganizationSettingsPaths,
} from "./organizationSettingsPaths";

function Paths({ organizationId }: { organizationId: string }) {
  const paths = useOrganizationSettingsPaths(organizationId);
  return (
    <output>
      {paths.apiKeyDetail("key-1")} {paths.secretDetail("secret-1")}
    </output>
  );
}

describe("useOrganizationSettingsPaths", () => {
  it("uses legacy organization settings URLs outside a provider", () => {
    render(<Paths organizationId="org-1" />);
    expect(screen.getByText("/org-1/settings/api-keys/key-1 /org-1/settings/secrets/secret-1")).toBeInTheDocument();
  });

  it("uses factory settings URLs inside a provider", () => {
    const paths: OrganizationSettingsPaths = {
      apiKeys: "/org-1/workspaces/RF/settings/organization/api-keys",
      apiKeyDetail: (id) => `/org-1/workspaces/RF/settings/organization/api-keys/${id}`,
      secrets: "/org-1/workspaces/RF/settings/organization/secrets",
      secretDetail: (id) => `/org-1/workspaces/RF/settings/organization/secrets/${id}`,
    };

    render(
      <OrganizationSettingsPathsProvider paths={paths}>
        <Paths organizationId="org-1" />
      </OrganizationSettingsPathsProvider>,
    );

    expect(
      screen.getByText(
        "/org-1/workspaces/RF/settings/organization/api-keys/key-1 /org-1/workspaces/RF/settings/organization/secrets/secret-1",
      ),
    ).toBeInTheDocument();
  });
});
