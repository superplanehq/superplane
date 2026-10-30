import { useState } from "react";
import { useSwitchFactoryModelSource } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { LLMModelsSwitchDialog, LLMModelsSwitchTarget } from "./FactorySettingsLLMModelsSwitchPreview";
import { byokProviderProductName } from "./organizationLLMModelsCopy";

type ModelSourceDialog = { step: "choose" } | { step: "subscription" } | { step: "connect" } | LLMModelsSwitchDialog;

export function useModelSourceSwitch(
  organizationId: string,
  factoryId: string,
  onSaved: (target: LLMModelsSwitchTarget | "subscription") => void,
  savedDefaultOwnerId = "",
) {
  const [defaultOwnerId, setDefaultOwnerId] = useState(savedDefaultOwnerId);
  const [subscriptionError, setSubscriptionError] = useState<string | null>(null);
  const [dialog, setDialog] = useState<ModelSourceDialog>(null);
  const [switchedNotice, setSwitchedNotice] = useState<string | null>(null);
  const switchSource = useSwitchFactoryModelSource(organizationId, factoryId);

  const saveSwitch = async (
    target: LLMModelsSwitchTarget | "subscription",
    connection?: { apiKey?: string; baseUrl?: string; apiType?: string },
  ) => {
    try {
      await switchSource.mutateAsync({
        source: target,
        ...(target === "subscription" ? { defaultOwnerId } : {}),
        apiKey: connection?.apiKey,
        ...(target === "custom" ? { baseUrl: connection?.baseUrl, apiType: connection?.apiType } : {}),
      });
      onSaved(target);
      setDialog(null);
      setSwitchedNotice(
        target === "subscription"
          ? "Automations in this workspace now use Codex subscriptions."
          : target === "hosted"
            ? "Automations in this workspace now use the SuperPlane agent."
            : target === "custom"
              ? "Automations in this workspace now use your custom provider."
              : `Automations in this workspace now use the ${byokProviderProductName(target)} agent.`,
      );
      showSuccessToast("Model source saved.");
      return true;
    } catch (switchError) {
      const message = getApiErrorMessage(switchError, "Unable to switch the model source.");
      if (target === "subscription") setSubscriptionError(message);
      showErrorToast(message);
      return false;
    }
  };

  return {
    dialog,
    setDialog,
    openDialog: (step: "choose" | "subscription" = "choose") => {
      setDefaultOwnerId(savedDefaultOwnerId);
      setSubscriptionError(null);
      setDialog({ step });
    },
    closeDialog: () => {
      setSubscriptionError(null);
      setDialog(null);
    },
    defaultOwnerId,
    setDefaultOwnerId: (ownerId: string) => {
      setSubscriptionError(null);
      setDefaultOwnerId(ownerId);
    },
    subscriptionError,
    switchedNotice,
    saveSwitch,
    saving: switchSource.isPending,
  };
}
