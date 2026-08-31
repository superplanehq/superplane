import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  answerCreateWithAgentSurvey,
  createCreateWithAgentDraft,
  runningCreateWithAgentView,
  sendCreateWithAgentMessage,
  setCreateWithAgentComposer,
} from "./createWithAgentDemo";

describe("createWithAgentDemo", () => {
  it("posts a survey after the first user message", () => {
    let view = runningCreateWithAgentView();
    view = setCreateWithAgentComposer(view, "Add a health check.");
    view = sendCreateWithAgentMessage(view);

    expect(view.composer).toBe("");
    expect(view.messages.some((message) => message.kind === "survey")).toBe(true);
    expect(view.right.kind).toBe("empty");
  });

  it("opens a draft after a survey answer and records a created task", () => {
    let view = runningCreateWithAgentView();
    view = setCreateWithAgentComposer(view, "Add a health check.");
    view = sendCreateWithAgentMessage(view);
    const survey = view.messages.find((message) => message.kind === "survey");
    if (!survey || survey.kind !== "survey") {
      throw new Error("expected a survey");
    }

    view = answerCreateWithAgentSurvey(view, survey.survey.id, [{ id: "area", value: "Payments" }]);
    expect(view.right.kind).toBe("draft");
    if (view.right.kind !== "draft") {
      return;
    }
    expect(view.right.draft.title).toContain("payments");

    view = createCreateWithAgentDraft(view);
    expect(view.right.kind).toBe("list");
    expect(view.created).toHaveLength(1);
    expect(view.created[0]?.key).toBe("NEW-1");
    expect(view.messages.at(-1)).toMatchObject({ kind: "text", text: CREATE_WITH_AGENT_COPY.afterCreate });
  });
});
