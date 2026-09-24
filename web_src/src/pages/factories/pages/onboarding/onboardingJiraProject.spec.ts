import { beforeEach, describe, expect, it } from "bun:test";

import { readOnboardingJiraProject, writeOnboardingJiraProject } from "./onboardingJiraProject";

describe("onboardingJiraProject", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores the selected project for a factory and Jira connection", () => {
    writeOnboardingJiraProject("factory-1", "jira-1", "PAY");

    expect(readOnboardingJiraProject("factory-1", "jira-1")).toBe("PAY");
    expect(readOnboardingJiraProject("factory-1", "jira-2")).toBe("");
  });

  it("clears the stored project when the selection is empty", () => {
    writeOnboardingJiraProject("factory-1", "jira-1", "PAY");
    writeOnboardingJiraProject("factory-1", "jira-1", "");

    expect(readOnboardingJiraProject("factory-1", "jira-1")).toBe("");
  });
});
