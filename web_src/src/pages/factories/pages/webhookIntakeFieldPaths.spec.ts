import { describe, expect, it } from "bun:test";

import {
  DEFAULT_WEBHOOK_INTAKE_FIELD_PATHS,
  webhookIntakeExampleBody,
  webhookIntakeFieldPathErrors,
} from "./webhookIntakeFieldPaths";

describe("webhookIntakeFieldPathErrors", () => {
  it("requires a title path and accepts empty description and link paths", () => {
    expect(webhookIntakeFieldPathErrors({ title: " ", description: "", link: "" })).toEqual({
      title: expect.any(String),
    });
    expect(webhookIntakeFieldPathErrors({ title: "issue.title", description: "", link: "" })).toEqual({});
  });

  it("rejects paths the server cannot read", () => {
    const errors = webhookIntakeFieldPathErrors({ title: "body[0]", description: "a..b", link: "issue.html-url" });
    expect(Object.keys(errors).sort()).toEqual(["description", "title"]);
  });
});

describe("webhookIntakeExampleBody", () => {
  it("nests each mapped field under its path", () => {
    const body = JSON.parse(
      webhookIntakeExampleBody({ title: "issue.title", description: "issue.body", link: "issue.url" }),
    );
    expect(Object.keys(body)).toEqual(["issue"]);
    expect(Object.keys(body.issue)).toEqual(["title", "body", "url"]);
  });

  it("uses top-level keys for the default paths", () => {
    expect(Object.keys(JSON.parse(webhookIntakeExampleBody(DEFAULT_WEBHOOK_INTAKE_FIELD_PATHS)))).toEqual([
      "title",
      "description",
      "url",
    ]);
  });
});
