import { useCreateFactoryPRFeedbackHandler, useFactoryRepositoryReviewBots } from "@/hooks/useFactoryPRFeedbackData";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useState } from "react";

import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackSource } from "./prFeedbackSettingsModel";
import { normalizeReviewBotLogin, type ReviewBotOption } from "./ReviewBotPicker";

export const DISCUSSION_MENTION = "@superplaneagent";

export function catalogReviewBots(catalog: Array<{ login?: string; displayName?: string }>): ReviewBotOption[] {
  const seen = new Set<string>();
  const bots: ReviewBotOption[] = [];
  for (const bot of catalog) {
    const login = bot.login?.trim();
    if (!login) {
      continue;
    }
    const key = login.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    bots.push({ login, displayName: bot.displayName?.trim() || login });
  }
  return bots;
}

export function useDiscussionPRFeedbackSetup(
  organizationId: string,
  factoryId: string,
  repository: string,
  open: boolean,
) {
  const [step, setStep] = useState<"mention" | "bots">("mention");
  const [mentionRequired, setMentionRequired] = useState(true);
  const [allowedBots, setAllowedBots] = useState<string[]>([]);
  const [error, setError] = useState<string>();
  const [catalogApplied, setCatalogApplied] = useState(false);

  const catalogQuery = useFactoryRepositoryReviewBots(organizationId, factoryId, repository, { enabled: open });
  const catalog = catalogReviewBots(catalogQuery.data ?? []);
  const catalogLoading = catalogQuery.isPending || catalogQuery.isFetching;
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);

  useEffect(() => {
    if (!open) {
      return;
    }
    setStep("mention");
    setMentionRequired(true);
    setAllowedBots([]);
    setError(undefined);
    setCatalogApplied(false);
  }, [open]);

  useEffect(() => {
    if (!open || catalogApplied || catalogLoading) {
      return;
    }
    setAllowedBots(catalog.map((bot) => bot.login));
    setCatalogApplied(true);
  }, [catalog, catalogApplied, catalogLoading, open]);

  const toggleBot = (login: string) => {
    setAllowedBots((current) => toggleUniqueString(current, login));
  };

  const addBot = (login: string): boolean => {
    const normalized = normalizeReviewBotLogin(login);
    if (!normalized) {
      return false;
    }
    setAllowedBots((current) => {
      if (current.some((item) => item.toLowerCase() === normalized.toLowerCase())) {
        return current;
      }
      return [...current, normalized];
    });
    return true;
  };

  const finish = async (source: PRFeedbackSource) => {
    setError(undefined);
    try {
      return await createHandler.mutateAsync({
        source: "SOURCE_PULL_REQUEST_DISCUSSION",
        name: source.defaultName,
        settings: {
          subject: repository ? { repository } : undefined,
          discussion: {
            mention: mentionRequired ? DISCUSSION_MENTION : "",
            ignoreBots: true,
            allowedBots,
          },
        },
      });
    } catch (cause) {
      setError(getApiErrorMessage(cause, PR_FEEDBACK_SETTINGS_COPY.createError));
      return undefined;
    }
  };

  return {
    step,
    setStep,
    mentionRequired,
    setMentionRequired,
    allowedBots,
    toggleBot,
    addBot,
    catalog,
    catalogQuery,
    catalogLoading,
    error,
    createHandler,
    finish,
  };
}

export type DiscussionPRFeedbackSetupModel = ReturnType<typeof useDiscussionPRFeedbackSetup>;
