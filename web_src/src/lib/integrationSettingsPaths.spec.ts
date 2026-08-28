import { describe, expect, it } from "vitest";

import {
  integrationDetailPath,
  integrationSetupPath,
  legacySettingsIntegrationsPath,
  organizationIntegrationsPath,
} from "./integrationSettingsPaths";

describe("integrationSettingsPaths", () => {
  it("builds the old settings catalog path and the factories organization path", () => {
    expect(legacySettingsIntegrationsPath("org-1")).toBe("/org-1/settings/integrations");
    expect(organizationIntegrationsPath("org-1")).toBe("/org-1/organization/integrations");
  });

  it("appends setup and detail segments onto a catalog base path", () => {
    const base = organizationIntegrationsPath("org-1");
    expect(integrationSetupPath(base, "github")).toBe("/org-1/organization/integrations/github/setup");
    expect(integrationDetailPath(base, "int-9")).toBe("/org-1/organization/integrations/int-9");
  });
});
