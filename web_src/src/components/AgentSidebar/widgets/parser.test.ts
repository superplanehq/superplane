import { describe, it, expect } from "vitest";
import { parseAgentContent } from "./parser";

describe("parseAgentContent", () => {
  it("parses pure markdown", () => {
    const content = "Hello **world**";
    const segments = parseAgentContent(content);
    expect(segments).toEqual([{ type: "markdown", content: "Hello **world**" }]);
  });

  it("parses buttons block", () => {
    const content = `:::buttons
- Option A
- Option B
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "buttons",
      items: ["Option A", "Option B"],
    });
  });

  it("parses confirm block", () => {
    const content = `:::confirm
message: Are you sure?
yes: Yes
no: No
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "confirm",
      message: "Are you sure?",
      yes: "Yes",
      no: "No",
    });
  });

  it("parses steps block", () => {
    const content = `:::steps
- [x] Done step
- [ ] Pending step
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "steps",
      items: [
        { done: true, text: "Done step" },
        { done: false, text: "Pending step" },
      ],
    });
  });

  it("parses success banner", () => {
    const content = `:::success
All good!
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "success",
      content: "All good!",
    });
  });

  it("parses mixed content", () => {
    const content = `Here is some text.

:::buttons
- Button 1
- Button 2
:::

More text here.`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(3);
    expect(segments[0].type).toBe("markdown");
    expect(segments[1].type).toBe("buttons");
    expect(segments[2].type).toBe("markdown");
  });

  it("handles chart JSON", () => {
    const content = `:::chart
{"type":"line","data":[{"x":1,"y":2}],"xKey":"x","yKeys":["y"]}
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "chart",
      config: {
        type: "line",
        data: [{ x: 1, y: 2 }],
        xKey: "x",
        yKeys: ["y"],
      },
    });
  });

  it("handles collapse block", () => {
    const content = `:::collapse
title: Click to expand
content: Hidden content
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(1);
    expect(segments[0]).toEqual({
      type: "collapse",
      title: "Click to expand",
      content: "Hidden content",
    });
  });

  it("handles empty content", () => {
    const segments = parseAgentContent("");
    expect(segments).toEqual([]);
  });

  it("handles block at start", () => {
    const content = `:::success
Great!
:::
Some text.`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(2);
    expect(segments[0].type).toBe("success");
    expect(segments[1].type).toBe("markdown");
  });

  it("handles block at end", () => {
    const content = `Some text.
:::error
Oops!
:::`;
    const segments = parseAgentContent(content);
    expect(segments).toHaveLength(2);
    expect(segments[0].type).toBe("markdown");
    expect(segments[1].type).toBe("error");
  });
});
