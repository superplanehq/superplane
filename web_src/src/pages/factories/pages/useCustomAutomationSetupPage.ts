import {
  useCreateFactoryCustomAutomation,
  useFactoryCustomAutomation,
  useUpdateFactoryCustomAutomation,
} from "@/hooks/useFactoryCustomAutomationData";
import { getApiErrorMessage } from "@/lib/errors";
import { useCustomAutomationEditorAgent } from "@/hooks/useCustomAutomationEditorAgent";
import { useEffect, useMemo, useState } from "react";

import type { FactoriesFactory, FactoriesFactoryAppTemplateAgent } from "@/api-client";
import {
  customAutomationColumn,
  customAutomationCredentials,
  customAutomationDraft,
  customAutomationRunsOnTask,
  customAutomationSettingsForTriggers,
  customAutomationTriggers,
  customAutomationRunContext,
  customAutomationTriggersForRun,
  DEFAULT_CUSTOM_AUTOMATION_ICON,
  newCustomAutomationTrigger,
  withCustomAutomationMergeStep,
  type CustomAutomationColumn,
  type CustomAutomationTrigger,
} from "../lib/customAutomation";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function useCustomAutomationSetupPage(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory?: FactoriesFactory;
  automationId?: string;
  expectedLiveVersionId?: string;
  columnKey?: CustomAutomationColumn;
}) {
  const described = useFactoryCustomAutomation(args.organizationId, args.factoryId, args.automationId);
  const create = useCreateFactoryCustomAutomation(args.organizationId, args.factoryId);
  const update = useUpdateFactoryCustomAutomation(args.organizationId, args.factoryId, args.automationId ?? "");
  const [name, setName] = useState("Custom automation");
  const [triggers, setTriggers] = useState<CustomAutomationTrigger[]>(() => [
    defaultCustomAutomationTrigger(args.columnKey),
  ]);
  const [runsOnTask, setRunsOnTask] = useState(true);
  const [icon, setIcon] = useState(DEFAULT_CUSTOM_AUTOMATION_ICON);
  const [edited, setDraft] = useState<PlanningReviewDraft>(() => customAutomationDraft({ settings: {} }));
  const [saveError, setSaveError] = useState("");
  const automation = described.data;
  const { agent, customProvider, ready } = useCustomAutomationEditorAgent(
    args.organizationId,
    args.factory?.onboarding,
    args.automationId,
    automation?.settings?.agent,
  );
  const draft = useMemo(() => withAgent(edited, agent, customProvider), [edited, agent, customProvider]);
  const columnKey = customAutomationColumn(automation?.automation?.columnKey ?? args.columnKey);

  useEffect(() => {
    if (args.automationId) {
      return;
    }
    setName("Custom automation");
    setTriggers([defaultCustomAutomationTrigger(args.columnKey)]);
    setRunsOnTask(true);
    setIcon(DEFAULT_CUSTOM_AUTOMATION_ICON);
    setDraft(customAutomationDraft({ settings: {} }));
    setSaveError("");
  }, [args.automationId, args.columnKey]);

  useEffect(() => {
    if (!automation) {
      return;
    }
    const savedTriggers = customAutomationTriggers(automation);
    setName(automation.automation?.name?.trim() || "Custom automation");
    setTriggers(savedTriggers);
    setRunsOnTask(customAutomationRunsOnTask(automation.settings?.runsOnTask, savedTriggers));
    setIcon(automation.settings?.icon?.trim() || DEFAULT_CUSTOM_AUTOMATION_ICON);
    setDraft(customAutomationDraft(automation));
  }, [automation]);

  const validationError = useMemo(() => validateDraft(name, draft), [draft, name]);
  const runContext = customAutomationRunContext(columnKey, triggers, runsOnTask);

  const save = async () => {
    const triggerError = validateTriggers(customAutomationTriggersForRun(triggers, runContext));
    if (triggerError) {
      setSaveError(triggerError);
      return false;
    }
    if (validationError) {
      setSaveError(validationError);
      return;
    }
    setSaveError("");
    const input = {
      name: name.trim(),
      settings: customAutomationSettingsForTriggers(triggers, draft, icon, runContext),
    };
    try {
      if (args.automationId) {
        await update.mutateAsync({ ...input, expectedLiveVersionId: args.expectedLiveVersionId });
      } else {
        await create.mutateAsync({ ...input, columnKey });
      }
      return true;
    } catch (error) {
      setSaveError(getApiErrorMessage(error, saveFallback(Boolean(args.automationId))));
      return false;
    }
  };

  return {
    columnKey,
    described,
    showModel: ready,
    name,
    setName,
    triggers,
    setTriggers: (next: CustomAutomationTrigger[]) => {
      setTriggers(next);
      setDraft((current) => withCustomAutomationMergeStep(current, next));
    },
    runsOnTask: runContext,
    setRunsOnTask,
    icon,
    setIcon,
    draft,
    setDraft,
    saveError,
    saving: create.isPending || update.isPending,
    save,
  };
}

function withAgent(
  draft: PlanningReviewDraft,
  agent: FactoriesFactoryAppTemplateAgent | undefined,
  customProvider: boolean,
) {
  return {
    ...draft,
    components: draft.components.map((component) => ({
      ...component,
      component: agent?.component,
      configuration: {
        ...component.configuration,
        credentials: customAutomationCredentials(agent),
        llmProvider: customProvider ? "custom" : undefined,
      },
    })),
  };
}

function defaultCustomAutomationTrigger(column: CustomAutomationColumn | undefined): CustomAutomationTrigger {
  return newCustomAutomationTrigger(column && column !== "verify" ? "automatic" : "manual");
}

function saveFallback(editing: boolean): string {
  return editing
    ? "SuperPlane could not save the automation. Try again."
    : "SuperPlane could not create the automation. Try again.";
}

function validateTriggers(triggers: CustomAutomationTrigger[]): string {
  if (triggers.length === 0) return "Add at least one trigger.";
  const invalidSchedule = triggers.some(
    (trigger) =>
      trigger.mode === "scheduled" &&
      (!trigger.schedule.cronExpression.trim() ||
        !Number.isFinite(trigger.schedule.utcOffset) ||
        trigger.schedule.utcOffset < -12 ||
        trigger.schedule.utcOffset > 14),
  );
  if (invalidSchedule) return "Enter a cron expression and choose a timezone.";
  if (triggers.some((trigger) => trigger.mode === "webhook" && !trigger.webhookId)) {
    return "Select a webhook for each webhook trigger.";
  }
  const incompleteIntegration = triggers.some(
    (trigger) =>
      trigger.mode === "integration" && (!trigger.integration?.integrationId || !trigger.integration.trigger),
  );
  return incompleteIntegration ? "Select an integration and an event for each integration trigger." : "";
}

function validateDraft(name: string, draft: PlanningReviewDraft): string {
  if (!name.trim()) {
    return "Enter an automation name.";
  }
  const rawSteps = draft.components[0]?.configuration.steps;
  const steps = Array.isArray(rawSteps) ? (rawSteps as PlanningReviewStep[]) : [];
  if (steps.length === 0) {
    return "Add at least one automation step.";
  }
  const invalid = steps.find(
    (step) => !step.name.trim() || (step.type === "prompt" ? !step.prompt?.trim() : !step.command?.trim()),
  );
  return invalid ? "Add a name and content to each automation step." : "";
}
