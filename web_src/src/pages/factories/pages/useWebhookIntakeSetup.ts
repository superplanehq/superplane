import { useIntakeOwnerDraft } from "./useIntakeOwnerDraft";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useFactoryWebhookMutations } from "@/hooks/useFactoryWebhooks";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import type { IntakeSetupStep } from "./IntakeSetupWizard";
import type { Authentication, Reveal } from "./settings/FactorySettingsWebhooksPage";
import {
  DEFAULT_WEBHOOK_INTAKE_FIELD_PATHS,
  webhookIntakeFieldPathErrors,
  type WebhookIntakeFieldPaths,
} from "./webhookIntakeFieldPaths";
import { WEBHOOK_INTAKE_SETUP_COPY } from "./webhookIntakeSetupCopy";

export type WebhookIntakeMode = "new" | "existing";

/** Finish result: the dialog shows the secret of a new webhook before it closes. */
export type WebhookIntakeFinish = "created" | "reveal" | "failed";

export function useWebhookIntakeSetup(organizationId: string, factoryId: string) {
  const [step, setStep] = useState<IntakeSetupStep>("connection");
  const [mode, setMode] = useState<WebhookIntakeMode>("new");
  const [name, setName] = useState<string>(WEBHOOK_INTAKE_SETUP_COPY.defaultName);
  const [authentication, setAuthentication] = useState<Authentication>("AUTHENTICATION_BEARER");
  const [webhookId, setWebhookId] = useState("");
  const [paths, setPaths] = useState<WebhookIntakeFieldPaths>(DEFAULT_WEBHOOK_INTAKE_FIELD_PATHS);
  // A webhook made by an earlier finish that failed to create the intake.
  const [createdWebhook, setCreatedWebhook] = useState<Reveal | null>(null);
  const [reveal, setReveal] = useState<Reveal | null>(null);
  const [error, setError] = useState<string>();

  const { create: createWebhook } = useFactoryWebhookMutations(organizationId, factoryId);
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const owner = useIntakeOwnerDraft(organizationId, factoryId);
  const pending = createWebhook.isPending || createIntake.isPending;
  const canContinue = mode === "new" ? name.trim() !== "" : webhookId !== "";
  const canFinish = Object.keys(webhookIntakeFieldPathErrors(paths)).length === 0 && !pending;

  const ensureWebhook = async (): Promise<Reveal | string> => {
    if (mode === "existing") return webhookId;
    if (createdWebhook) return createdWebhook;
    const created = await createWebhook.mutateAsync({ name: name.trim(), authentication, headerName: "" });
    if (!created?.webhook?.id || !created.secret) {
      throw new Error(WEBHOOK_INTAKE_SETUP_COPY.webhookCreateError);
    }
    const next = { webhook: created.webhook, secret: created.secret };
    setCreatedWebhook(next);
    return next;
  };

  const finish = async (): Promise<WebhookIntakeFinish> => {
    setError(undefined);
    try {
      const webhook = await ensureWebhook();
      await createIntake.mutateAsync({
        source: "SOURCE_WEBHOOK",
        resourceId: typeof webhook === "string" ? webhook : webhook.webhook.id,
        settings: {
          ...owner.settings(),
          webhookTitlePath: paths.title.trim(),
          webhookDescriptionPath: paths.description.trim(),
          webhookLinkPath: paths.link.trim(),
        },
      });
      if (typeof webhook === "string") return "created";
      setReveal(webhook);
      return "reveal";
    } catch (cause) {
      setError(getApiErrorMessage(cause, WEBHOOK_INTAKE_SETUP_COPY.wizardCreateError));
      return "failed";
    }
  };

  return {
    owner,
    step,
    setStep,
    mode,
    setMode,
    name,
    setName,
    authentication,
    setAuthentication,
    webhookId,
    setWebhookId,
    paths,
    setPaths,
    createdWebhook,
    reveal,
    error,
    pending,
    canContinue,
    canFinish,
    finish,
  };
}

export type WebhookIntakeSetupModel = ReturnType<typeof useWebhookIntakeSetup>;
