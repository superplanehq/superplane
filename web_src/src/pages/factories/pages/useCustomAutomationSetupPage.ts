import {
  useCreateFactoryCustomAutomation,
  useFactoryCustomAutomation,
  useUpdateFactoryCustomAutomation,
} from "@/hooks/useFactoryCustomAutomationData";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import {
  customAutomationDraft,
  customAutomationMode,
  customAutomationSettings,
  customAutomationSchedule,
  DEFAULT_CUSTOM_AUTOMATION_ICON,
  workspaceAgentFromOnboarding,
  type CustomAutomationMode,
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
}) {
  const described = useFactoryCustomAutomation(args.organizationId, args.factoryId, args.automationId);
  const create = useCreateFactoryCustomAutomation(args.organizationId, args.factoryId);
  const update = useUpdateFactoryCustomAutomation(args.organizationId, args.factoryId, args.automationId ?? "");
  const [name, setName] = useState("Custom automation");
  const [mode, setMode] = useState<CustomAutomationMode>("manual");
  const [schedule, setSchedule] = useState(() => customAutomationSchedule());
  const [icon, setIcon] = useState(DEFAULT_CUSTOM_AUTOMATION_ICON);
  const agentHarness = args.factory?.onboarding?.agentHarness;
  const [draft, setDraft] = useState<PlanningReviewDraft>(() =>
    customAutomationDraft({ settings: { agent: workspaceAgentFromOnboarding({ agentHarness }) } }),
  );
  const [saveError, setSaveError] = useState("");
  const automation = described.data;

  useEffect(() => {
    if (args.automationId) {
      return;
    }
    setName("Custom automation");
    setMode("manual");
    setSchedule(customAutomationSchedule());
    setIcon(DEFAULT_CUSTOM_AUTOMATION_ICON);
    setDraft(customAutomationDraft({ settings: { agent: workspaceAgentFromOnboarding({ agentHarness }) } }));
    setSaveError("");
  }, [args.automationId, agentHarness]);

  useEffect(() => {
    if (!automation) {
      return;
    }
    setName(automation.automation?.name?.trim() || "Custom automation");
    setMode(customAutomationMode(automation.settings?.startMode));
    setSchedule(customAutomationSchedule(automation));
    setIcon(automation.settings?.icon?.trim() || DEFAULT_CUSTOM_AUTOMATION_ICON);
    setDraft(customAutomationDraft(automation));
  }, [automation]);

  const boardHref = factoryHomePath(args.organizationId, args.factoryKey, firstFactoryLineId(args.factory));
  const line = args.factory?.lines?.find((entry) => entry.id === args.lineId);
  const returnHref = line?.id ? factoryLineDetailPath(args.organizationId, args.factoryKey, line.id) : boardHref;
  const validationError = useMemo(() => validateDraft(name, draft), [draft, name]);

  const save = async () => {
    if (
      mode === "scheduled" &&
      (!schedule.cronExpression.trim() ||
        !Number.isFinite(schedule.utcOffset) ||
        schedule.utcOffset < -12 ||
        schedule.utcOffset > 14)
    ) {
      setSaveError("Enter a cron expression and a UTC offset from -12 to +14 hours.");
      return false;
    }
    if (validationError) {
      setSaveError(validationError);
      return;
    }
    setSaveError("");
    const input = {
      name: name.trim(),
      settings: customAutomationSettings(mode, draft, icon, schedule),
    };
    try {
      if (args.automationId) {
        await update.mutateAsync(input);
      } else {
        await create.mutateAsync(input);
      }
      return true;
    } catch (error) {
      setSaveError(getApiErrorMessage(error, saveFallback(Boolean(args.automationId))));
      return false;
    }
  };

  return {
    boardHref,
    returnHref,
    lineExists: Boolean(line),
    described,
    name,
    setName,
    mode,
    setMode,
    schedule,
    setSchedule,
    icon,
    setIcon,
    draft,
    setDraft,
    saveError,
    saving: create.isPending || update.isPending,
    save,
  };
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
