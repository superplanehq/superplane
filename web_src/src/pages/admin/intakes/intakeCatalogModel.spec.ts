import { describe, expect, it } from "bun:test";

import { INTAKE_STATUSES } from "@/lib/intakeCatalog";

import {
  INTAKE_STATUS_INFO,
  intakeAccessSummary,
  intakeKeyFromName,
  MATURITY_STEPS,
  widensAccess,
  type AdminIntakeEntry,
} from "./intakeCatalogModel";

const entry = (overrides: Partial<AdminIntakeEntry> = {}): AdminIntakeEntry => ({
  key: "datadog",
  name: "Datadog errors",
  category: "error_tracking",
  status: "beta",
  status_note: "",
  enabled_for_all: false,
  implemented: true,
  deletable: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  updated_by_name: "",
  organizations: [],
  ...overrides,
});

describe("INTAKE_STATUS_INFO", () => {
  it("explains every status", () => {
    for (const status of INTAKE_STATUSES) {
      const info = INTAKE_STATUS_INFO[status];
      expect(info.label).not.toBe("");
      expect(info.summary).not.toBe("");
      expect(info.whoCanUse).not.toBe("");
    }
  });

  it("tells when to move on for every step before Generally available", () => {
    for (const status of MATURITY_STEPS.filter((step) => step !== "ga")) {
      expect(INTAKE_STATUS_INFO[status].nextStep).not.toBe("");
    }
  });
});

describe("intakeAccessSummary", () => {
  it("names the companies that can use the intake", () => {
    expect(intakeAccessSummary(entry({ implemented: false, status: "planned" }))).toBe("Not implemented");
    expect(intakeAccessSummary(entry({ status: "ga" }))).toBe("All companies");
    expect(intakeAccessSummary(entry({ enabled_for_all: true }))).toBe("All companies");
    expect(intakeAccessSummary(entry())).toBe("No companies");
    const added = { id: "org-1", name: "Acme", added_at: new Date().toISOString() };
    expect(intakeAccessSummary(entry({ organizations: [added] }))).toBe("1 company");
    expect(intakeAccessSummary(entry({ status: "deprecated", organizations: [added] }))).toBe("No new intakes");
  });
});

describe("widensAccess", () => {
  it("asks for confirmation only when more companies get access", () => {
    expect(widensAccess("beta", "ga")).toBe(true);
    expect(widensAccess("deprecated", "beta")).toBe(true);
    expect(widensAccess("ga", "beta")).toBe(false);
    expect(widensAccess("beta", "deprecated")).toBe(false);
  });
});

describe("intakeKeyFromName", () => {
  it("makes a slug that the API accepts", () => {
    expect(intakeKeyFromName("  Azure DevOps Boards! ")).toBe("azure-devops-boards");
    expect(intakeKeyFromName("GitLab")).toBe("gitlab");
  });
});
