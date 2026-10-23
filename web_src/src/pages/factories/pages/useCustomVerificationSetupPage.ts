import {
  useCreateFactoryCustomVerification,
  useFactoryCustomVerification,
  useUpdateFactoryCustomVerification,
} from "@/hooks/useFactoryCustomVerificationData";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import {
  customVerificationDraft,
  customVerificationMode,
  customVerificationSettings,
  DEFAULT_CUSTOM_VERIFICATION_ICON,
  type CustomVerificationMode,
} from "../lib/customVerification";
import { factoryHomePath, factoryLineDetailPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function useCustomVerificationSetupPage(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory?: FactoriesFactory;
  lineId?: string;
  automationId?: string;
}) {
  const described = useFactoryCustomVerification(args.organizationId, args.factoryId, args.automationId);
  const create = useCreateFactoryCustomVerification(args.organizationId, args.factoryId);
  const update = useUpdateFactoryCustomVerification(args.organizationId, args.factoryId, args.automationId ?? "");
  const [name, setName] = useState("Custom automation");
  const [mode, setMode] = useState<CustomVerificationMode>("manual");
  const [icon, setIcon] = useState(DEFAULT_CUSTOM_VERIFICATION_ICON);
  const [draft, setDraft] = useState<PlanningReviewDraft>(() => customVerificationDraft());
  const [saveError, setSaveError] = useState("");
  const verification = described.data;

  useEffect(() => {
    if (args.automationId) {
      return;
    }
    setName("Custom automation");
    setMode("manual");
    setIcon(DEFAULT_CUSTOM_VERIFICATION_ICON);
    setDraft(customVerificationDraft());
    setSaveError("");
  }, [args.automationId]);

  useEffect(() => {
    if (!verification) {
      return;
    }
    setName(verification.automation?.name?.trim() || "Custom automation");
    setMode(customVerificationMode(verification.settings?.startMode));
    setIcon(verification.settings?.icon?.trim() || DEFAULT_CUSTOM_VERIFICATION_ICON);
    setDraft(customVerificationDraft(verification));
  }, [verification]);

  const boardHref = factoryHomePath(args.organizationId, args.factoryKey, firstFactoryLineId(args.factory));
  const line = args.factory?.lines?.find((entry) => entry.id === args.lineId);
  const returnHref = line?.id ? factoryLineDetailPath(args.organizationId, args.factoryKey, line.id) : boardHref;
  const validationError = useMemo(() => validateDraft(name, draft), [draft, name]);

  const save = async () => {
    if (validationError) {
      setSaveError(validationError);
      return;
    }
    setSaveError("");
    const input = {
      name: name.trim(),
      settings: customVerificationSettings(mode, draft, Boolean(args.automationId), icon),
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
