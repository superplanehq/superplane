import { describe, expect, it } from "bun:test";

import { seededIntakeCatalog } from "@/test/intakeCatalog";

import { addIntakeTemplatesFromCatalog } from "./addIntakeTemplates";

const ids = (templates: { id: string }[]) => templates.map((template) => template.id);

describe("addIntakeTemplatesFromCatalog", () => {
  it("lists available sources first, then Beta, then coming soon", () => {
    const templates = addIntakeTemplatesFromCatalog(seededIntakeCatalog(["datadog"]));

    expect(ids(templates)).toEqual(["github-issues", "dependabot-alerts", "sentry-exceptions", "datadog", "notion"]);
    expect(templates.find((template) => template.id === "datadog")).toMatchObject({ beta: true, soon: false });
    expect(ids(templates)).not.toContain("jira-issues");
  });

  it("keeps a Planned intake as coming soon when the company can see it", () => {
    const templates = addIntakeTemplatesFromCatalog(seededIntakeCatalog(["linear-issues"]));

    expect(templates.find((template) => template.id === "linear-issues")).toMatchObject({ soon: true, beta: false });
  });

  it("hides a Deprecated source and shows an Internal source when the company can see it", () => {
    const hidden = addIntakeTemplatesFromCatalog(
      seededIntakeCatalog([], {
        "sentry-exceptions": { status: "deprecated", available: false },
        "dependabot-alerts": { status: "alpha", available: false },
      }),
    );
    expect(ids(hidden)).not.toContain("sentry-exceptions");
    expect(ids(hidden)).not.toContain("dependabot-alerts");

    const internal = addIntakeTemplatesFromCatalog(
      seededIntakeCatalog([], { "dependabot-alerts": { status: "alpha", available: true } }),
    );
    expect(internal.find((template) => template.id === "dependabot-alerts")).toMatchObject({
      soon: false,
      beta: false,
    });
  });

  it("lists a source that an admin added without code as coming soon with the catalog name", () => {
    const templates = addIntakeTemplatesFromCatalog(
      seededIntakeCatalog([], {
        "azure-boards": { name: "Azure Boards", category: "issue_tracking", status: "planned" },
        gitea: { name: "Gitea", category: "repository_provider", status: "planned" },
      }),
    );

    expect(templates.find((template) => template.id === "azure-boards")).toMatchObject({
      name: "Azure Boards",
      soon: true,
      iconSrc: undefined,
    });
    expect(ids(templates)).not.toContain("gitea");
  });

  it("uses the catalog name over the name in code", () => {
    const templates = addIntakeTemplatesFromCatalog(
      seededIntakeCatalog([], { "github-issues": { name: "GitHub Issues (cloud)" } }),
    );

    expect(templates.find((template) => template.id === "github-issues")?.name).toBe("GitHub Issues (cloud)");
  });
});
