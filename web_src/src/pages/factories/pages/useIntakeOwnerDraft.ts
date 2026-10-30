import { useState } from "react";
import type { FactoriesFactoryIntakeSettings } from "@/api-client";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";

export function useIntakeOwnerDraft(organizationId: string, factoryId: string) {
  const layout = useOptionalFactoriesLayout();
  const required = layout?.factory?.onboarding?.codexSubscription === true;
  const scope = `${organizationId}:${factoryId}:${required}`;
  const [choice, setChoice] = useState<{ scope: string; value: string } | null>(null);
  const value =
    choice?.scope === scope ? choice.value : required ? (layout?.factory?.onboarding?.codexDefaultOwnerId ?? "") : "";
  const setValue = (next: string) => setChoice({ scope, value: next });
  return {
    organizationId,
    factoryId,
    required,
    value,
    setValue,
    settings: (): Pick<FactoriesFactoryIntakeSettings, "defaultOwnerId"> => {
      if (!required && !value) return {};
      if (required && !value) throw new Error("Select a default task owner with Codex connected.");
      return { defaultOwnerId: value };
    },
  };
}

export type IntakeOwnerDraft = ReturnType<typeof useIntakeOwnerDraft>;
