import { describe, expect, it } from "bun:test";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  formatPlanningSurveyReply,
  isPlanningSurveyReply,
  parsePlanningSurvey,
  parsePlanningSurveyReply,
} from "./planningSessionSurvey";

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

describe("parsePlanningSurveyReply", () => {
  it("splits each line into the question and the chosen answer", () => {
    expect(
      parsePlanningSurveyReply("What is the priority? High\nWhat is the scope? One file"),
    ).toEqual([
      { question: "What is the priority?", answer: "High" },
      { question: "What is the scope?", answer: "One file" },
    ]);
  });

  it("keeps a long custom answer after the question mark", () => {
    expect(
      parsePlanningSurveyReply(
        "What should the agent build? Custom styled modal matching the app dark theme, per the maintainer comment",
      ),
    ).toEqual([
      {
        question: "What should the agent build?",
        answer: "Custom styled modal matching the app dark theme, per the maintainer comment",
      },
    ]);
  });

  it("returns no pairs for the skip line", () => {
    expect(parsePlanningSurveyReply(CREATE_WITH_AGENT_COPY.surveySkipped)).toEqual([]);
  });
});

describe("isPlanningSurveyReply", () => {
  it("matches formatted answers and the skip line", () => {
    expect(isPlanningSurveyReply("What is the priority? High\nWhat is the scope? skipped")).toBe(true);
    expect(isPlanningSurveyReply(CREATE_WITH_AGENT_COPY.surveySkipped)).toBe(true);
    expect(isPlanningSurveyReply("I want to add color to puppies")).toBe(false);
  });
});
