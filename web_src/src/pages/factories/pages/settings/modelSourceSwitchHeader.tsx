import { DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import type { LLMModelsSwitchDialog, LLMModelsSwitchProvider } from "./FactorySettingsLLMModelsSwitchPreview";
import { modelSourceSwitchLabel } from "./organizationLLMModelsCopy";

export function renderModelSourceSwitchHeader(
  source: "hosted" | "subscription" | LLMModelsSwitchProvider,
  dialog: LLMModelsSwitchDialog,
) {
  if (!dialog) return null;
  const next = modelSourceSwitchLabel(dialog.target);
  if (dialog.step === "key") {
    return (
      <DialogHeader>
        <DialogTitle>{dialog.target === "custom" ? "Custom provider" : `Your ${next} key`}</DialogTitle>
        <DialogDescription>
          {dialog.target === "custom"
            ? "Set the provider URL, token, and API type. SuperPlane stores the token for agents in this workspace."
            : "SuperPlane stores this key for agents in this workspace."}
        </DialogDescription>
      </DialogHeader>
    );
  }
  return (
    <DialogHeader>
      <DialogTitle>Switch automations to {next}?</DialogTitle>
      <DialogDescription>
        This replaces the {modelSourceSwitchLabel(source)} agent in every automation in this workspace with the {next}{" "}
        agent. {dialog.target === "hosted" ? "SuperPlane bills these runs." : `${next} bills these runs.`}
      </DialogDescription>
    </DialogHeader>
  );
}
