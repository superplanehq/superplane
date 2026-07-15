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
      prompt: "",
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
    const content = `:::collapse title="Click to expand"
Hidden content
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

  it("keeps normal multiple-choice survey options together", () => {
    const segments = parseAgentContent(`:::survey
Which environment should we deploy to?
- Staging
- Production
- [input]
:::`);

    expect(segments).toEqual([
      {
        type: "survey",
        questions: [
          {
            prompt: "Which environment should we deploy to?",
            options: ["Staging", "Production"],
            hasInput: true,
          },
        ],
      },
    ]);
  });

  it("keeps multiple-choice survey question options together", () => {
    const segments = parseAgentContent(`:::survey
Choose the question to ask next:
- Did the agent finish analyzing the PR?
- Was a report generated for the run?
- Is the download command being run on the same PR?
:::`);

    expect(segments).toEqual([
      {
        type: "survey",
        questions: [
          {
            prompt: "Choose the question to ask next:",
            options: [
              "Did the agent finish analyzing the PR?",
              "Was a report generated for the run?",
              "Is the download command being run on the same PR?",
            ],
          },
        ],
      },
    ]);
  });

  it("treats question-like survey bullet items as separate free-text questions", () => {
    const segments = parseAgentContent(`:::survey
**When you tested \`/download-report\`:**
- Did you first create a PR to trigger the agent and let it finish analyzing?
- Are you trying to use \`/download-report\` on a PR that never had an agent run on it?
- Did you wait for the agent to complete before commenting \`/download-report\`?
:::`);

    expect(segments).toEqual([
      {
        type: "survey",
        questions: [
          {
            prompt: "Did you first create a PR to trigger the agent and let it finish analyzing?",
            options: [],
            hasInput: true,
          },
          {
            prompt: "Are you trying to use `/download-report` on a PR that never had an agent run on it?",
            options: [],
            hasInput: true,
          },
          {
            prompt: "Did you wait for the agent to complete before commenting `/download-report`?",
            options: [],
            hasInput: true,
          },
        ],
      },
    ]);
  });

  it("treats markdown-emphasized question bullet items as separate free-text questions", () => {
    const segments = parseAgentContent(`:::survey
**When you tested \`/download-report\`:**
- **Did you first create a PR to trigger the agent and let it finish analyzing?**
- **Did you wait for the agent to complete before commenting \`/download-report\`?**
:::`);

    expect(segments).toEqual([
      {
        type: "survey",
        questions: [
          {
            prompt: "**Did you first create a PR to trigger the agent and let it finish analyzing?**",
            options: [],
            hasInput: true,
          },
          {
            prompt: "**Did you wait for the agent to complete before commenting `/download-report`?**",
            options: [],
            hasInput: true,
          },
        ],
      },
    ]);
  });

  it("preserves full markdown bodies inside categorized rubric sections", () => {
    const content = `:::rubric HTTP Monitor Spec
## Flow
1. **Schedule** fires every 15 minutes
2. **HTTP GET** \`https://httpbin.org/get\`

## Components
| Node | Component |
|------|-----------|
| Schedule | \`schedule\` |

\`\`\`yaml
minutesInterval: 15
\`\`\`
:::`;

    const segments = parseAgentContent(content);

    expect(segments).toHaveLength(1);
    expect(segments[0]).toMatchObject({
      type: "rubric",
      title: "HTTP Monitor Spec",
      body: expect.stringContaining("```yaml"),
      categories: [
        {
          heading: "Flow",
          body: expect.stringContaining("**Schedule**"),
        },
        {
          heading: "Components",
          body: expect.stringContaining("| Node | Component |"),
        },
      ],
    });
  });
});
