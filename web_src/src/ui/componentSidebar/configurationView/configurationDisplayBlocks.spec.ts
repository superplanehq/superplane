import { describe, expect, it } from "vitest";
import { isNestedGroupHeader, parseConfigurationDisplayBlocks } from "./configurationDisplayBlocks";
import type { ConfigurationDisplayRow } from "./types";

describe("parseConfigurationDisplayBlocks", () => {
  it("identifies structural group headers by key suffix", () => {
    expect(
      isNestedGroupHeader({ key: "authConfig.__group", label: "Auth", kind: "text", displayText: "2 items" }),
    ).toBe(true);
    expect(isNestedGroupHeader({ key: "headers[0].__header", label: "Header 1", kind: "text", displayText: "" })).toBe(
      true,
    );
    expect(
      isNestedGroupHeader({ key: "environment", label: "Environment", kind: "text", displayText: "Production" }),
    ).toBe(false);
  });

  it("groups nested object fields under a single block", () => {
    const rows: ConfigurationDisplayRow[] = [
      { key: "authConfig.__group", label: "Authentication", kind: "text", displayText: "", depth: 0 },
      { key: "authConfig.authMethod", label: "Auth method", kind: "text", displayText: "API token", depth: 1 },
      { key: "authConfig.token", label: "Token", kind: "text", displayText: "••••••", depth: 1 },
      { key: "environment", label: "Environment", kind: "text", displayText: "Production", depth: 0 },
    ];

    const blocks = parseConfigurationDisplayBlocks(rows);
    expect(blocks).toHaveLength(2);
    expect(blocks[0]).toMatchObject({
      type: "group",
      header: { key: "authConfig.__group" },
    });
    expect(blocks[0].type === "group" && blocks[0].children).toHaveLength(2);
    expect(blocks[1]).toMatchObject({ type: "row", row: { key: "environment" } });
  });

  it("nests list item groups with their own vertical grouping", () => {
    const rows: ConfigurationDisplayRow[] = [
      { key: "headers.__group", label: "Headers", kind: "list", displayText: "2 items", depth: 0 },
      { key: "headers[0].__header", label: "Header 1", kind: "text", displayText: "", depth: 1 },
      { key: "headers[0].key", label: "Key", kind: "text", displayText: "X-Environment", depth: 2 },
      { key: "headers[0].value", label: "Value", kind: "text", displayText: "production", depth: 2 },
      { key: "headers[1].__header", label: "Header 2", kind: "text", displayText: "", depth: 1 },
      { key: "headers[1].key", label: "Key", kind: "text", displayText: "X-Request-Source", depth: 2 },
    ];

    const blocks = parseConfigurationDisplayBlocks(rows);
    expect(blocks).toHaveLength(1);
    expect(blocks[0].type).toBe("group");

    if (blocks[0].type !== "group") {
      return;
    }

    expect(blocks[0].children).toHaveLength(2);
    expect(blocks[0].children[0].type).toBe("group");
    expect(blocks[0].children[1].type).toBe("group");
  });
});
