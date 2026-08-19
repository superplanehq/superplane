import { describe, expect, it } from "vitest";
import { NOTIFICATION_TYPE_OPTIONS } from "./FactorySettingsNotificationsPage";

describe("NOTIFICATION_TYPE_OPTIONS", () => {
  it("states each event label without requiring the tooltip", () => {
    const labelsByKey = Object.fromEntries(NOTIFICATION_TYPE_OPTIONS.map((option) => [option.key, option.label]));

    expect(labelsByKey).toEqual({
      TYPE_WORK_ORDER_ASSIGNED: "Added as a work order owner",
      TYPE_WORK_ORDER_COMMENT_OWNED: "Comments on work orders you own",
      TYPE_WORK_ORDER_COMMENT_CREATED: "Comments on work orders you created",
      TYPE_WORK_ORDER_STATUS_OWNED: "Status changes on work orders you own or created",
      TYPE_WORK_ORDER_ARTIFACT_OWNED: "New artifacts on work orders you own",
      TYPE_WORK_ORDER_MENTIONED: "Mentions in work order comments",
    });
  });

  it("keeps the two comment labels parallel so neither reads as authorship", () => {
    const owned = NOTIFICATION_TYPE_OPTIONS.find((option) => option.key === "TYPE_WORK_ORDER_COMMENT_OWNED");
    const created = NOTIFICATION_TYPE_OPTIONS.find((option) => option.key === "TYPE_WORK_ORDER_COMMENT_CREATED");

    expect(owned?.label.startsWith("Comments on work orders you ")).toBe(true);
    expect(created?.label.startsWith("Comments on work orders you ")).toBe(true);
  });
});
