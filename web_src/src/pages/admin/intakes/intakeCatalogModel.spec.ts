import { describe, expect, it } from "bun:test";

import { INTAKE_STATUSES } from "@/lib/intakeCatalog";

import { INTAKE_STATUS_INFO, intakeKeyFromName, MATURITY_STEPS } from "./intakeCatalogModel";

describe("INTAKE_STATUS_INFO", () => {
  it("explains every status", () => {
    for (const status of INTAKE_STATUSES) {
      const info = INTAKE_STATUS_INFO[status];
      expect(info.label).not.toBe("");
      expect(info.summary).not.toBe("");
    }
  });

  it("tells when to move on for every step before Generally available", () => {
    for (const status of MATURITY_STEPS.filter((step) => step !== "ga")) {
      expect(INTAKE_STATUS_INFO[status].nextStep).not.toBe("");
    }
  });
});

describe("intakeKeyFromName", () => {
  it("makes a slug that the API accepts", () => {
    expect(intakeKeyFromName("  Azure DevOps Boards! ")).toBe("azure-devops-boards");
    expect(intakeKeyFromName("GitLab")).toBe("gitlab");
  });

  it("does not return a trailing dash after truncation", () => {
    expect(intakeKeyFromName(`${"a".repeat(63)} Z`)).toBe("a".repeat(63));
  });
});
