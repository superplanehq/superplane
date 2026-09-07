import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  createCreateWithAgentDraft,
  runningCreateWithAgentView,
  sendCreateWithAgentMessage,
  setCreateWithAgentComposer,
} from "./createWithAgentDemo";

describe("createWithAgentDemo", () => {
  it("keeps the user message after send", () => {
    let view = runningCreateWithAgentView();
    view = setCreateWithAgentComposer(view, "Add a health check.");
    view = sendCreateWithAgentMessage(view);

    expect(view.composer).toBe("");
    expect(view.messages.some((message) => message.kind === "text" && message.text === "Add a health check.")).toBe(
      true,
    );
    expect(view.right.kind).toBe("empty");
  });

  it("records a created task from a draft", () => {
    let view = runningCreateWithAgentView({
      right: { kind: "draft", draft: { title: "Improve payments", description: "Stop double charges." } },
    });

    view = createCreateWithAgentDraft(view);
    expect(view.right.kind).toBe("empty");
    expect(view.created).toHaveLength(1);
    expect(view.created[0]?.key).toBe("NEW-1");
    expect(view.messages.at(-1)).toMatchObject({ kind: "text", text: CREATE_WITH_AGENT_COPY.afterCreate });
  });
});
