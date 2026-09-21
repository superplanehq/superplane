import { describe, expect, it } from "bun:test";

import { preferredJiraCompletionColumn } from "./JiraCompletionColumnFields";

describe("preferredJiraCompletionColumn", () => {
  it("keeps the current column when it is still available", () => {
    expect(preferredJiraCompletionColumn(["To Do", "QA", "Done"], "QA")).toBe("QA");
  });

  it("prefers Done when the current column is empty", () => {
    expect(preferredJiraCompletionColumn(["To Do", "In Progress", "Done"], "")).toBe("Done");
  });

  it("uses the first column when Done is missing", () => {
    expect(preferredJiraCompletionColumn(["Ready", "Closed"], "")).toBe("Ready");
  });
});
