import { describe, expect, it } from "vitest";
import {
  factoryAppConfigurePath,
  factoryAppPath,
  factoryAppRunPath,
  factoryAppSplitRunPath,
  factoryAppViewPath,
  factoryDetailPath,
  factoryHomePath,
  pathAfterWorkspaceSwitch,
  factoryIntakePath,
  factoryPRFeedbackPath,
  intakeSettingsTabFromSearch,
  intakeIdFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  prFeedbackHandlerIdFromSearch,
  prFeedbackSettingsTabFromSearch,
  factorySettingsGeneralPathAfterKeyChange,
  factorySettingsSectionPath,
  factorySettingsWorkspaceGeneralPath,
  replaceOrganizationSegment,
  createWorkOrderPath,
  firstFactoryLineId,
  firstFactoryLineName,
  legacyWorkOrderDetailPath,
  organizationSettingsBackPath,
  organizationSettingsPath,
  organizationSettingsSectionPath,
  parseFactoryAppNavFrom,
  workOrderDetailPath,
  workOrderOpenPath,
  workOrdersPath,
} from "./factoryPagePaths";

describe("factoryDetailPath", () => {
  it("builds the workspace URL from the workspace key", () => {
    expect(factoryDetailPath("org-1", "SP")).toBe("/org-1/workspaces/SP");
  });
});

describe("factoryHomePath", () => {
  it("opens the first line board when a line id is present", () => {
    expect(factoryHomePath("org-1", "SP", "line-plan")).toBe("/org-1/workspaces/SP/lines/line-plan");
  });

  it("opens the workspace index when no line id is present", () => {
    expect(factoryHomePath("org-1", "SP")).toBe("/org-1/workspaces/SP");
  });
});

describe("pathAfterWorkspaceSwitch", () => {
  const nextFactory = { key: "AO", lines: [{ id: "line-acme" }] };

  it("keeps the settings page", () => {
    expect(
      pathAfterWorkspaceSwitch({
        pathname: "/org-1/workspaces/RF/settings/workspace/general",
        organizationId: "org-1",
        currentFactoryKey: "RF",
        nextFactory,
      }),
    ).toBe("/org-1/workspaces/AO/settings/workspace/general");
  });

  it("keeps Velocity", () => {
    expect(
      pathAfterWorkspaceSwitch({
        pathname: "/org-1/workspaces/RF/velocity",
        organizationId: "org-1",
        currentFactoryKey: "RF",
        nextFactory,
      }),
    ).toBe("/org-1/workspaces/AO/velocity");
  });

  it("opens the new workspace board from a line that belongs to the previous workspace", () => {
    expect(
      pathAfterWorkspaceSwitch({
        pathname: "/org-1/workspaces/RF/lines/line-plan",
        organizationId: "org-1",
        currentFactoryKey: "RF",
        nextFactory,
      }),
    ).toBe("/org-1/workspaces/AO/lines/line-acme");
  });
});

describe("factoryIntakePath", () => {
  it("opens the line board with the intake query", () => {
    expect(factoryIntakePath("org-1", "SP", "line-plan")).toBe("/org-1/workspaces/SP/lines/line-plan?intake=1");
  });

  it("reads the intake query from the search string", () => {
    expect(isIntakeSearchOpen("?intake=1")).toBe(true);
    expect(isIntakeSearchOpen("intake=1")).toBe(true);
    expect(isIntakeSearchOpen("")).toBe(false);
  });

  it("opens the line board with a selected intake", () => {
    expect(factoryIntakePath("org-1", "SP", "line-plan", "intake-1")).toBe(
      "/org-1/workspaces/SP/lines/line-plan?intake=1&intakeId=intake-1",
    );
    expect(factoryIntakePath("org-1", "SP", "line-plan", "intake-1", "automation")).toBe(
      "/org-1/workspaces/SP/lines/line-plan?intake=1&intakeId=intake-1&settings=automation",
    );
  });

  it("reads the intake id from the search string", () => {
    expect(intakeIdFromSearch("?intake=1&intakeId=intake-1")).toBe("intake-1");
    expect(intakeIdFromSearch("intake=1")).toBeNull();
  });

  it("reads the settings tab from the search string", () => {
    expect(intakeSettingsTabFromSearch("?intake=1&settings=automation")).toBe("automation");
    expect(intakeSettingsTabFromSearch("intake=1")).toBeNull();
  });
});

describe("factoryPRFeedbackPath", () => {
  it("opens the line board with the PR feedback query", () => {
    expect(factoryPRFeedbackPath("org-1", "SP", "line-plan")).toBe("/org-1/workspaces/SP/lines/line-plan?prFeedback=1");
  });

  it("reads the PR feedback query from the search string", () => {
    expect(isPRFeedbackSearchOpen("?prFeedback=1")).toBe(true);
    expect(isPRFeedbackSearchOpen("prFeedback=1")).toBe(true);
    expect(isPRFeedbackSearchOpen("")).toBe(false);
  });

  it("opens the line board on a settings tab", () => {
    expect(factoryPRFeedbackPath("org-1", "SP", "line-plan", "automation")).toBe(
      "/org-1/workspaces/SP/lines/line-plan?prFeedback=1&prFeedbackSettings=automation",
    );
  });

  it("reads the settings tab from the search string", () => {
    expect(prFeedbackSettingsTabFromSearch("?prFeedback=1&prFeedbackSettings=automation")).toBe("automation");
    expect(prFeedbackSettingsTabFromSearch("prFeedback=1")).toBeNull();
  });

  it("opens a specific handler", () => {
    expect(factoryPRFeedbackPath("org-1", "SP", "line-plan", undefined, "handler-1")).toBe(
      "/org-1/workspaces/SP/lines/line-plan?prFeedback=1&prFeedbackHandler=handler-1",
    );
    expect(prFeedbackHandlerIdFromSearch("?prFeedback=1&prFeedbackHandler=handler-1")).toBe("handler-1");
  });
});

describe("firstFactoryLineId", () => {
  it("returns the first line that has an id", () => {
    expect(firstFactoryLineId({ lines: [{ id: "line-a" }, { id: "line-b" }] })).toBe("line-a");
  });

  it("returns undefined when the factory has no line", () => {
    expect(firstFactoryLineId({ lines: [] })).toBeUndefined();
  });
});

describe("firstFactoryLineName", () => {
  it("returns the first line that has a name", () => {
    expect(firstFactoryLineName({ lines: [{ name: "plan-and-implement" }, { name: "hotfix" }] })).toBe(
      "plan-and-implement",
    );
  });

  it("returns undefined when the factory has no named line", () => {
    expect(firstFactoryLineName({ lines: [{ name: "  " }] })).toBeUndefined();
  });
});

describe("workOrdersPath", () => {
  it("builds the tasks list URL", () => {
    expect(workOrdersPath("org-1", "SP")).toBe("/org-1/workspaces/SP/tasks");
  });
});

describe("createWorkOrderPath", () => {
  it("builds the create-task URL under the tasks list", () => {
    expect(createWorkOrderPath("org-1", "SP")).toBe("/org-1/workspaces/SP/tasks/new");
  });
});

describe("workOrderDetailPath", () => {
  it("builds the canonical permalink from the workspace key and task number", () => {
    expect(workOrderDetailPath("org-1", "SP", 42)).toBe("/org-1/workspaces/SP/task/42");
  });

  it("accepts the number as a string", () => {
    expect(workOrderDetailPath("org-1", "SP", "42")).toBe("/org-1/workspaces/SP/task/42");
  });

  it("is a sibling of, not nested under, the plural tasks list path", () => {
    expect(workOrderDetailPath("org-1", "SP", "42")).not.toContain(workOrdersPath("org-1", "SP"));
  });

  it("keeps the board line on the permalink when a line id is given", () => {
    expect(workOrderDetailPath("org-1", "SP", "42", "line-hotfix")).toBe(
      "/org-1/workspaces/SP/task/42?lineId=line-hotfix",
    );
  });
});

describe("workOrderOpenPath", () => {
  it("uses the canonical permalink when the order has a number", () => {
    expect(workOrderOpenPath("org-1", "SP", 42, "line-1")).toBe("/org-1/workspaces/SP/task/42");
  });

  it("falls back to the line board when the order has no number", () => {
    expect(workOrderOpenPath("org-1", "SP", undefined, "line-1")).toBe("/org-1/workspaces/SP/lines/line-1");
  });
});

describe("legacyWorkOrderDetailPath", () => {
  it("builds the old id-based shape for back-compat redirects", () => {
    expect(legacyWorkOrderDetailPath("org-1", "SP", "order-uuid")).toBe("/org-1/workspaces/SP/work-orders/order-uuid");
  });
});

describe("factoryAppPath", () => {
  it("encodes orderNumber (not orderId) in the query string", () => {
    expect(factoryAppPath("org-1", "SP", "app-1", { from: "task", orderNumber: "42" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?from=task&orderNumber=42",
    );
  });
});

describe("replaceOrganizationSegment", () => {
  it("keeps the settings path when switching organization", () => {
    expect(replaceOrganizationSegment("/demo/workspaces/RF/settings/organization/general", "demo", "acme")).toBe(
      "/acme/workspaces/RF/settings/organization/general",
    );
  });

  it("keeps the settings path when the current URL uses the organization id", () => {
    expect(
      replaceOrganizationSegment("/org-uuid/workspaces/RF/settings/organization/integrations", "org-uuid", "acme"),
    ).toBe("/acme/workspaces/RF/settings/organization/integrations");
  });

  it("opens the workspace list when the path is not under the current organization", () => {
    expect(replaceOrganizationSegment("/other/workspaces", "demo", "acme")).toBe("/acme/workspaces");
  });
});

describe("factorySettingsGeneralPathAfterKeyChange", () => {
  it("returns the General settings URL when the key changes", () => {
    expect(factorySettingsGeneralPathAfterKeyChange("org-1", "RF", "AB")).toBe(
      "/org-1/workspaces/AB/settings/workspace/general",
    );
  });

  it("returns null when the key does not change", () => {
    expect(factorySettingsGeneralPathAfterKeyChange("org-1", "RF", "RF")).toBeNull();
  });
});

describe("factorySettingsSectionPath", () => {
  it("builds a scoped settings URL", () => {
    expect(factorySettingsWorkspaceGeneralPath("org-1", "RF")).toBe("/org-1/workspaces/RF/settings/workspace/general");
    expect(factorySettingsSectionPath("org-1", "RF", "organization", "api-keys")).toBe(
      "/org-1/workspaces/RF/settings/organization/api-keys",
    );
  });
});

describe("factoryAppConfigurePath", () => {
  it("adds configure=1, opens the agent panel, and keeps the components panel closed", () => {
    expect(factoryAppConfigurePath("org-1", "SP", "app-1")).toBe("/org-1/workspaces/SP/apps/app-1?configure=1&agent=1");
  });

  it("keeps the run when entering edit from a run page", () => {
    expect(factoryAppConfigurePath("org-1", "SP", "app-1", { from: "lines", lineId: "line-1", runId: "run-9" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?run=run-9&configure=1&agent=1&from=lines&lineId=line-1",
    );
  });

  it("opens components only when blocks is requested", () => {
    expect(factoryAppConfigurePath("org-1", "SP", "app-1", { blocks: true })).toBe(
      "/org-1/workspaces/SP/apps/app-1?configure=1&agent=1&blocks=1",
    );
  });

  it("opens the component sidebar on the selected node", () => {
    expect(factoryAppConfigurePath("org-1", "SP", "app-1", { nodeId: "create-pr" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?configure=1&agent=1&sidebar=1&node=create-pr",
    );
  });

  it("does not keep run inspection when opening a component for edit", () => {
    expect(
      factoryAppConfigurePath("org-1", "SP", "app-1", {
        from: "lines",
        lineId: "line-1",
        runId: "run-9",
        nodeId: "create-pr",
      }),
    ).toBe("/org-1/workspaces/SP/apps/app-1?configure=1&agent=1&sidebar=1&node=create-pr&from=lines&lineId=line-1");
  });
});

describe("factoryAppSplitRunPath", () => {
  it("opens the split run page with canvas and line context", () => {
    expect(
      factoryAppSplitRunPath("org-1", "SP", "app-1", {
        from: "lines",
        lineId: "line-1",
        runId: "run-9",
        orderNumber: "103",
        canvas: "implementation",
      }),
    ).toBe(
      "/org-1/workspaces/SP/apps/app-1/split-run?run=run-9&from=lines&lineId=line-1&orderNumber=103&canvas=implementation",
    );
  });
});

describe("factoryAppViewPath", () => {
  it("opens the canvas run inspector when a run id is present", () => {
    expect(factoryAppViewPath("org-1", "SP", "app-1", { from: "lines", lineId: "line-1", runId: "run-9" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?run=run-9&from=lines&lineId=line-1",
    );
  });
});

describe("factoryAppRunPath", () => {
  it("opens the canvas run inspector", () => {
    expect(factoryAppRunPath("org-1", "SP", "app-1", "run-9", { from: "lines", lineId: "line-1" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?run=run-9&from=lines&lineId=line-1",
    );
  });
});

describe("parseFactoryAppNavFrom", () => {
  it("accepts known from values", () => {
    expect(parseFactoryAppNavFrom("lines")).toBe("lines");
    expect(parseFactoryAppNavFrom("task")).toBe("task");
  });

  it("normalizes the legacy work-order value to task", () => {
    expect(parseFactoryAppNavFrom("work-order")).toBe("task");
  });

  it("returns undefined for unknown from values", () => {
    expect(parseFactoryAppNavFrom("canvas")).toBeUndefined();
  });
});

describe("organizationSettingsPath", () => {
  it("builds organization settings under the organization, not a workspace", () => {
    expect(organizationSettingsPath("org-1")).toBe("/org-1/organization");
    expect(organizationSettingsSectionPath("org-1", "general")).toBe("/org-1/organization/general");
  });

  it("returns to the workspace when settings opened from a factory, otherwise the list", () => {
    expect(organizationSettingsBackPath("org-1", "RF")).toBe("/org-1/workspaces/RF");
    expect(organizationSettingsBackPath("org-1")).toBe("/org-1/workspaces");
  });
});
