import { describe, expect, it } from "vitest";
import { buildNodeNamesById } from "./nodeNamesById";

type Item = { id?: string; name?: string };

describe("buildNodeNamesById", () => {
  it("maps id to name for each item", () => {
    const items: Item[] = [
      { id: "node-1", name: "Fetch Data" },
      { id: "node-2", name: "Validate Schema" },
    ];

    const result = buildNodeNamesById(
      items,
      (item) => item.id,
      (item) => item.name,
    );

    expect(result).toEqual({ "node-1": "Fetch Data", "node-2": "Validate Schema" });
  });

  it("skips an item missing either field", () => {
    const items: Item[] = [{ id: "node-1", name: "Fetch Data" }, { id: "node-2" }, { name: "Orphaned Name" }];

    const result = buildNodeNamesById(
      items,
      (item) => item.id,
      (item) => item.name,
    );

    expect(result).toEqual({ "node-1": "Fetch Data" });
  });
});
