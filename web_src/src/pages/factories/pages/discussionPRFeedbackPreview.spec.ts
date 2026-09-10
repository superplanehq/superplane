import { describe, expect, it } from "vitest";

import {
  discussionMentionPreviewComment,
  discussionPreviewBotAuthors,
  discussionPreviewBotInitials,
  discussionPreviewBotIsSkipped,
  discussionSetupPreviewCaption,
} from "./discussionPRFeedbackPreview";
import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";
import { DISCUSSION_MENTION } from "./useDiscussionPRFeedbackSetup";

describe("discussionMentionPreviewComment", () => {
  it("includes the SuperPlane mention when a mention is required", () => {
    expect(discussionMentionPreviewComment(true)).toBe(
      `${DISCUSSION_MENTION} ${PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentBody}`,
    );
  });

  it("omits the SuperPlane mention when any human comment starts a run", () => {
    expect(discussionMentionPreviewComment(false)).toBe(PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentBody);
  });
});

describe("discussionSetupPreviewCaption", () => {
  it("describes the mention rule on the human-comments step", () => {
    expect(discussionSetupPreviewCaption("mention", true, "ignore")).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCaptionRequire,
    );
    expect(discussionSetupPreviewCaption("mention", false, "address")).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCaptionAny,
    );
  });

  it("describes the bot rule on the AI-comments step", () => {
    expect(discussionSetupPreviewCaption("bots", true, "ignore")).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotIgnoreCaption,
    );
    expect(discussionSetupPreviewCaption("bots", false, "address", 2)).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotAddressCaption,
    );
    expect(discussionSetupPreviewCaption("bots", false, "address", 0)).toBe(
      PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotIgnoreCaption,
    );
  });
});

describe("discussionPreviewBotAuthors", () => {
  const catalog = [
    { login: "coderabbitai", displayName: "coderabbitai[bot]" },
    { login: "bugbot", displayName: "bugbot[bot]" },
  ];

  it("uses the selected bots when SuperPlane addresses bot comments", () => {
    expect(discussionPreviewBotAuthors("address", ["bugbot", "coderabbitai"], catalog)).toEqual([
      { login: "bugbot", displayName: "bugbot[bot]" },
      { login: "coderabbitai", displayName: "coderabbitai[bot]" },
    ]);
  });

  it("keeps a manual bot login when it is not in the catalog", () => {
    expect(discussionPreviewBotAuthors("address", ["my-reviewer"], catalog)).toEqual([
      { login: "my-reviewer", displayName: "my-reviewer" },
    ]);
  });

  it("uses an example bot when none are selected", () => {
    expect(discussionPreviewBotAuthors("address", [], catalog)).toEqual([catalog[0]]);
    expect(discussionPreviewBotIsSkipped("address", [])).toBe(true);
  });

  it("uses the first catalog bot as the ignored example", () => {
    expect(discussionPreviewBotAuthors("ignore", ["bugbot"], catalog)).toEqual([catalog[0]]);
    expect(discussionPreviewBotIsSkipped("ignore", ["bugbot"])).toBe(true);
    expect(discussionPreviewBotIsSkipped("address", ["bugbot"])).toBe(false);
  });
});

describe("discussionPreviewBotInitials", () => {
  it("uses the first two letters of the bot login", () => {
    expect(discussionPreviewBotInitials("coderabbitai")).toBe("CO");
    expect(discussionPreviewBotInitials("bugbot")).toBe("BU");
  });
});
