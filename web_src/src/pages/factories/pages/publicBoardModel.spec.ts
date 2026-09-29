import { describe, expect, it } from "bun:test";

import { EMPTY_WORK_ORDER_FILTERS } from "../lib/workOrderListModel";
import {
  legacyAssigneeChipOptions,
  visibleBoardColumns,
  type PublicBoardCard,
  type PublicBoardColumn,
} from "./publicBoardModel";

function card(name: string, key: string): PublicBoardCard {
  return { title: name, createdAt: "2026-01-01T00:00:00Z", assignee: { name, key } };
}

function column(cards: PublicBoardCard[]): PublicBoardColumn {
  return { key: "backlog", title: "Backlog", cards };
}

describe("legacy owner filters", () => {
  it("keeps every person who still uses the saved name", () => {
    const columns = [column([card("Ada Lovelace", "aaaaaaaaaaaaaaaa"), card("Ada Lovelace", "bbbbbbbbbbbbbbbb")])];
    const filters = { ...EMPTY_WORK_ORDER_FILTERS, assigneeIds: ["public-member:ada lovelace"] };

    const visible = visibleBoardColumns(columns, filters, "");

    expect(visible[0]?.cards).toHaveLength(2);
  });

  it("labels the saved name without replacing it", () => {
    const savedId = "public-member:ada lovelace";
    const options = legacyAssigneeChipOptions(
      [column([card("Ada Lovelace", "aaaaaaaaaaaaaaaa")])],
      [savedId, "public-member:aaaaaaaaaaaaaaaa"],
    );

    expect(options).toEqual([{ value: savedId, label: "Ada Lovelace" }]);
  });
});
