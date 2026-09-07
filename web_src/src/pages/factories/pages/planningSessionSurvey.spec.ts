import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { formatPlanningSurveyReply, isPlanningSurveyReply, parsePlanningSurvey } from "./planningSessionSurvey";

describe("parsePlanningSurvey", () => {
  it("reads questions from survey JSON", () => {
    expect(
      parsePlanningSurvey(
        JSON.stringify({
          questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
        }),
      ),
    ).toEqual({
      questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
    });
  });

  it("returns undefined for empty or invalid survey JSON", () => {
    expect(parsePlanningSurvey("")).toBeUndefined();
    expect(parsePlanningSurvey("{")).toBeUndefined();
    expect(parsePlanningSurvey(JSON.stringify({ questions: [] }))).toBeUndefined();
  });
});

describe("formatPlanningSurveyReply", () => {
  const questions = [
    { prompt: "What is the priority?", options: ["High", "Low"] },
    { prompt: "What is the scope?", options: ["One file", "The service"] },
  ];

  it("formats picked and skipped answers", () => {
    expect(formatPlanningSurveyReply(questions, ["High", null])).toBe(
      "What is the priority? High\nWhat is the scope? skipped",
    );
  });

  it("uses the skip line when every question is empty", () => {
    expect(formatPlanningSurveyReply(questions, [null, ""])).toBe(CREATE_WITH_AGENT_COPY.surveySkipped);
  });
});

describe("isPlanningSurveyReply", () => {
  it("matches formatted answers and the skip line", () => {
    expect(isPlanningSurveyReply("What is the priority? High\nWhat is the scope? skipped")).toBe(true);
    expect(isPlanningSurveyReply(CREATE_WITH_AGENT_COPY.surveySkipped)).toBe(true);
    expect(isPlanningSurveyReply("I want to add color to puppies")).toBe(false);
  });
});
