import { describe, expect, it } from "bun:test";

import type { EventInfo } from "../types";
import { onPRCommentTriggerRenderer } from "./on_pr_comment";

describe("onPRCommentTriggerRenderer.getRootEventValues", () => {
  it("returns all seven event fields in display order with the PR HTML URL", () => {
    const event = prCommentEvent({
      comment: {
        user: { login: "octocat" },
        body: "Ready for review.",
        html_url: "https://github.com/acme/project/pull/42#issuecomment-1",
      },
      issue: {
        number: 42,
        title: "Update the integration",
        pull_request: {
          html_url: "https://github.com/acme/project/pull/42",
          url: "https://api.github.com/repos/acme/project/pulls/42",
        },
      },
    });

    const values = onPRCommentTriggerRenderer.getRootEventValues({ event });

    expect(Object.keys(values)).toEqual([
      "Received At",
      "Author",
      "Comment Body",
      "Comment URL",
      "PR Number",
      "PR Title",
      "PR URL",
    ]);
    expect(values).toEqual({
      "Received At": new Date(event.createdAt).toLocaleString(),
      Author: "octocat",
      "Comment Body": "Ready for review.",
      "Comment URL": "https://github.com/acme/project/pull/42#issuecomment-1",
      "PR Number": "42",
      "PR Title": "Update the integration",
      "PR URL": "https://github.com/acme/project/pull/42",
    });
  });

  it("uses fallbacks for missing comment and issue sections", () => {
    const event = prCommentEvent({});

    expect(onPRCommentTriggerRenderer.getRootEventValues({ event })).toEqual({
      "Received At": new Date(event.createdAt).toLocaleString(),
      Author: "-",
      "Comment Body": "-",
      "Comment URL": "-",
      "PR Number": "-",
      "PR Title": "-",
      "PR URL": "-",
    });
  });

  it("uses all seven fallbacks when the event is absent", () => {
    expect(onPRCommentTriggerRenderer.getRootEventValues({ event: undefined })).toEqual({
      "Received At": "-",
      Author: "-",
      "Comment Body": "-",
      "Comment URL": "-",
      "PR Number": "-",
      "PR Title": "-",
      "PR URL": "-",
    });
  });

  it.each([
    {
      html_url: undefined,
      url: "https://api.github.com/repos/acme/project/pulls/42",
      expected: "https://api.github.com/repos/acme/project/pulls/42",
    },
    {
      html_url: "",
      url: "https://api.github.com/repos/acme/project/pulls/42",
      expected: "https://api.github.com/repos/acme/project/pulls/42",
    },
    { html_url: undefined, url: undefined, expected: "-" },
    { html_url: "", url: "", expected: "-" },
  ])("uses $expected when PR URL candidates are $html_url and $url", ({ html_url, url, expected }) => {
    const event = prCommentEvent({
      issue: {
        html_url: "https://github.com/acme/project/issues/42",
        pull_request: { html_url, url },
      },
    });

    expect(onPRCommentTriggerRenderer.getRootEventValues({ event })["PR URL"]).toBe(expected);
  });

  it("uses fallbacks for empty strings while preserving PR number zero", () => {
    const event = prCommentEvent({
      comment: { user: { login: "" }, body: "", html_url: "" },
      issue: { number: 0, title: "" },
    });
    event.createdAt = "";

    expect(onPRCommentTriggerRenderer.getRootEventValues({ event })).toEqual({
      "Received At": "-",
      Author: "-",
      "Comment Body": "-",
      "Comment URL": "-",
      "PR Number": "0",
      "PR Title": "-",
      "PR URL": "-",
    });
  });
});

function prCommentEvent(data: Record<string, unknown>): NonNullable<EventInfo> {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    nodeId: "node-1",
    type: "github.prComment",
    data,
  };
}
