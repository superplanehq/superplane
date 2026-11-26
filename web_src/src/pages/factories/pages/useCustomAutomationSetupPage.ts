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
  type CustomAutomationColumn,
  type CustomAutomationTrigger,
} from "../lib/customAutomation";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function useCustomAutomationSetupPage(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory?: FactoriesFactory;
  lineId?: string;
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

  const boardHref = factoryHomePath(args.organizationId, args.factoryKey, firstFactoryLineId(args.factory));
  const line = args.factory?.lines?.find((entry) => entry.id === args.lineId);
  const returnHref = line?.id ? factoryLineDetailPath(args.organizationId, args.factoryKey, line.id) : boardHref;
  const validationError = useMemo(() => validateDraft(name, draft), [draft, name]);
  const runContext = customAutomationRunContext(columnKey, triggers, runsOnTask);

  const save = async () => {
    const activeTriggers = customAutomationTriggersForRun(triggers, runContext);
    const invalidSchedule = activeTriggers.some(
      (trigger) =>
        trigger.mode === "scheduled" &&
        (!trigger.schedule.cronExpression.trim() ||
          !Number.isFinite(trigger.schedule.utcOffset) ||
          trigger.schedule.utcOffset < -12 ||
          trigger.schedule.utcOffset > 14),
    );
    if (invalidSchedule) {
      setSaveError("Enter a cron expression and choose a timezone.");
      return false;
    }
    if (activeTriggers.some((trigger) => trigger.mode === "webhook" && !trigger.webhookId)) {
      setSaveError("Select a webhook for each webhook trigger.");
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
    boardHref,
    returnHref,
    lineExists: Boolean(line),
    described,
    showModel: ready,
    name,
    setName,
    triggers,
    setTriggers,
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
