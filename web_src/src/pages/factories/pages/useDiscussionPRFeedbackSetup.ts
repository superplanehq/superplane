import { useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useState } from "react";

import {
  PR_FEEDBACK_SETTINGS_COPY,
  toggleUniqueString,
  type PRFeedbackDraftSettings,
  type PRFeedbackSource,
} from "./prFeedbackSettingsModel";
import { normalizeReviewBotLogin, type ReviewBotOption } from "./reviewBotPickerModel";

export const DISCUSSION_MENTION = "@superplaneagent";

export type DiscussionBotMode = "ignore" | "address";

export function catalogReviewBots(catalog: Array<{ id?: string; name?: string }>): ReviewBotOption[] {
  const seen = new Set<string>();
  const bots: ReviewBotOption[] = [];
  for (const bot of catalog) {
    const login = bot.id?.trim();
    if (!login) {
      continue;
    }
    const key = login.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    bots.push({ login, displayName: bot.name?.trim() || login });
  }
  return bots;
}

export function discussionBotModeFromDraft(
  draft: Pick<PRFeedbackDraftSettings, "ignoreBots" | "allowedBots">,
): DiscussionBotMode {
  if (!draft.ignoreBots || draft.allowedBots.length > 0) {
    return "address";
  }
  return "ignore";
}

export function discussionBotSettings(
  mode: DiscussionBotMode,
  allowedBots: string[],
): { ignoreBots: boolean; allowedBots: string[] } {
  if (mode === "address") {
    return { ignoreBots: true, allowedBots };
  }
  return { ignoreBots: true, allowedBots: [] };
}

export function useDiscussionPRFeedbackSetup(
  organizationId: string,
  factoryId: string,
  githubIntegrationId: string,
  repository: string,
) {
  const [step, setStep] = useState<"mention" | "bots">("mention");
  const [mentionRequired, setMentionRequired] = useState(true);
  const [botMode, setBotMode] = useState<DiscussionBotMode>("ignore");
  const [allowedBots, setAllowedBots] = useState<string[]>([]);
  const [error, setError] = useState<string>();
  const [catalogApplied, setCatalogApplied] = useState(false);

  const catalogParameters = repository.trim() ? { repository: repository.trim() } : undefined;
  const catalogQuery = useIntegrationResources(organizationId, githubIntegrationId, "review_bot", catalogParameters);
  const catalog = catalogReviewBots(catalogQuery.data ?? []);
  const catalogLoading = catalogQuery.isPending || catalogQuery.isFetching;
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);

  useEffect(() => {
    if (catalogApplied || catalogLoading) {
      return;
    }
    setAllowedBots(catalog.map((bot) => bot.login));
    setCatalogApplied(true);
  }, [catalog, catalogApplied, catalogLoading]);

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
      const bots = discussionBotSettings(botMode, allowedBots);
      return await createHandler.mutateAsync({
        source: "SOURCE_PULL_REQUEST_DISCUSSION",
        name: source.defaultName,
        settings: {
          subject: repository ? { repository } : undefined,
          discussion: {
            mention: mentionRequired ? DISCUSSION_MENTION : "",
            ignoreBots: bots.ignoreBots,
            allowedBots: bots.allowedBots,
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
    botMode,
    setBotMode,
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
