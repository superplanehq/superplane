import { describe, expect, it } from "bun:test";

import type { CreateWithAgentTaskMessage } from "../createWithAgentTypes";
import { createdTaskHref } from "./workOrderPopupActions";

const TASK: CreateWithAgentTaskMessage = {
  id: "task-1",
  kind: "task",
  role: "task",
  workOrderId: "wo-2",
  key: "NEW-2",
  title: "Add the retry table",
  number: 2,
};

describe("createdTaskHref", () => {
  it("links a created task to its permalink on the same board line", () => {
    expect(createdTaskHref("org-1", "acme", "line-1")(TASK)).toBe("/org-1/workspaces/acme/task/2?lineId=line-1");
    expect(createdTaskHref("org-1", "acme")(TASK)).toBe("/org-1/workspaces/acme/task/2");
  });

  it("returns nothing without factory context or a task number", () => {
    expect(createdTaskHref(undefined, "acme")(TASK)).toBeUndefined();
    expect(createdTaskHref("org-1", undefined)(TASK)).toBeUndefined();
    expect(createdTaskHref("org-1", "acme")({ ...TASK, number: undefined })).toBeUndefined();
  });
});
