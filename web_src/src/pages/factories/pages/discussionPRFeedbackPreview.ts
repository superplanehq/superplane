import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";
import type { ReviewBotOption } from "./reviewBotPickerModel";
import { DISCUSSION_MENTION, type DiscussionBotMode } from "./useDiscussionPRFeedbackSetup";

export function discussionMentionPreviewComment(mentionRequired: boolean): string {
  if (mentionRequired) {
    return `${DISCUSSION_MENTION} ${PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentBody}`;
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentBody;
}

export function discussionSetupPreviewCaption(
  step: "mention" | "bots",
  mentionRequired: boolean,
  botMode: DiscussionBotMode,
  selectedBotCount = 0,
): string {
  if (step === "bots") {
    if (botMode === "address" && selectedBotCount > 0) {
      return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotAddressCaption;
    }
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotIgnoreCaption;
  }
  return mentionRequired
    ? PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCaptionRequire
    : PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCaptionAny;
}

export function discussionPreviewBotIsSkipped(botMode: DiscussionBotMode, selected: string[]): boolean {
  return botMode === "ignore" || !selected.some((login) => login.trim().length > 0);
}

function discussionPreviewExampleBot(catalog: ReviewBotOption[]): ReviewBotOption {
  if (catalog[0]) {
    return catalog[0];
  }
  const login = PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotAuthor;
  return { login, displayName: login };
}

export function discussionPreviewBotAuthors(
  botMode: DiscussionBotMode,
  selected: string[],
  catalog: ReviewBotOption[],
): ReviewBotOption[] {
  if (botMode === "address") {
    const authors = selectedPreviewBots(selected, catalog);
    if (authors.length > 0) {
      return authors;
    }
  }
  return [discussionPreviewExampleBot(catalog)];
}

function selectedPreviewBots(selected: string[], catalog: ReviewBotOption[]): ReviewBotOption[] {
  const catalogByLogin = new Map<string, ReviewBotOption>();
  for (const bot of catalog) {
    catalogByLogin.set(bot.login.trim().toLowerCase(), bot);
  }

  const authors: ReviewBotOption[] = [];
  const seen = new Set<string>();
  for (const login of selected) {
    const trimmed = login.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    authors.push(catalogByLogin.get(key) ?? { login: trimmed, displayName: trimmed });
  }
  return authors;
}

export function discussionPreviewBotInitials(login: string): string {
  const cleaned = login.replace(/\[bot\]$/i, "").replace(/[^a-z0-9]+/gi, "");
  if (cleaned.length >= 2) {
    return cleaned.slice(0, 2).toUpperCase();
  }
  if (cleaned.length === 1) {
    return cleaned.toUpperCase();
  }
  return "?";
}
