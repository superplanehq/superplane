import { useState } from "react";
import type { FactoriesFactoryIntakeSettings } from "@/api-client";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";

export function useIntakeOwnerDraft(organizationId: string, factoryId: string) {
  const layout = useOptionalFactoriesLayout();
  const required = layout?.factory?.onboarding?.codexSubscription === true;
  const [value, setValue] = useState("");
  return {
    organizationId,
    factoryId,
    required,
    value,
    setValue,
    settings: (): Pick<FactoriesFactoryIntakeSettings, "defaultOwnerId"> => {
      if (!required) return {};
      if (!value) throw new Error("Select a default task owner with Codex connected.");
      return { defaultOwnerId: value };
    },
  };
}

export type IntakeOwnerDraft = ReturnType<typeof useIntakeOwnerDraft>;
