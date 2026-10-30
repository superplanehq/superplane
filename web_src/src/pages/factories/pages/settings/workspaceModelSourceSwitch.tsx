import type { useModelSourceSwitch } from "./useModelSourceSwitch";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { IntakeDefaultOwnerField } from "../IntakeDefaultOwnerField";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import { type LLMModelsSwitchProvider, SwitchDialog } from "./FactorySettingsLLMModelsSwitchPreview";
import { byokProviderProductName } from "./organizationLLMModelsCopy";
import type { WorkspaceModelSource } from "./workspaceModelSource";

export function ModelSourceSwitchDialogs({
  organizationId,
  factoryId,
  source,
  currentProvider,
  connected,
  providers,
  switcher,
}: {
  organizationId: string;
  factoryId: string;
  source: WorkspaceModelSource;
  currentProvider: LLMModelsSwitchProvider | null;
  connected: LLMModelsSwitchProvider[];
  providers: LLMModelsSwitchProvider[];
  switcher: ReturnType<typeof useModelSourceSwitch>;
}) {
  const eligibility = useFactoryOwnerEligibility(organizationId, factoryId, undefined, true);
  const selectedEligible =
    eligibility.data?.members?.some((member) => member.userId === switcher.defaultOwnerId && member.eligible) === true;
  return (
    <>
      <Dialog
        open={switcher.subscriptionDialog === "warn"}
        onOpenChange={(open) => {
          if (!open) switcher.setSubscriptionDialog(null);
        }}
      >
        <DialogContent aria-describedby={undefined}>
          <DialogHeader>
            <DialogTitle>Switch to Codex</DialogTitle>
          </DialogHeader>
          <IntakeDefaultOwnerField
            organizationId={organizationId}
            factoryId={factoryId}
            required
            prospectiveSubscription
            value={switcher.defaultOwnerId}
            onChange={switcher.setDefaultOwnerId}
            disabled={switcher.saving}
          />
          {switcher.subscriptionError ? <p role="alert">{switcher.subscriptionError}</p> : null}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => switcher.setSubscriptionDialog(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              disabled={!selectedEligible || switcher.saving || eligibility.isFetching}
              onClick={() => void switcher.saveSwitch("subscription")}
            >
              Switch to Codex
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <SwitchDialog
        source={source === "hosted" || source === "subscription" ? source : (currentProvider ?? "anthropic")}
        dialog={switcher.dialog}
        saving={switcher.saving}
        providerSelector={
          switcher.dialog?.target !== "hosted" ? (
            <Select
              value={switcher.dialog?.target ?? ""}
              onValueChange={(target) =>
                switcher.setDialog({ target: target as LLMModelsSwitchProvider, step: "warn" })
              }
              disabled={switcher.saving}
            >
              <SelectTrigger aria-label="New API key provider">
                <SelectValue placeholder="Select a provider" />
              </SelectTrigger>
              <SelectContent>
                {providers.map((provider) => (
                  <SelectItem key={provider} value={provider}>
                    {byokProviderProductName(provider)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : undefined
        }
        onCancel={() => switcher.setDialog(null)}
        onContinueToKey={() => {
          if (!switcher.dialog || switcher.dialog.target === "hosted") {
            return;
          }
          if (connected.includes(switcher.dialog.target)) {
            void switcher.saveSwitch(switcher.dialog.target);
            return;
          }
          switcher.setDialog({ target: switcher.dialog.target, step: "key" });
        }}
        onBackToWarning={() => {
          if (switcher.dialog) {
            switcher.setDialog({ target: switcher.dialog.target, step: "warn" });
          }
        }}
        onSaveKey={(apiKey) => {
          if (switcher.dialog && switcher.dialog.target !== "hosted") {
            void switcher.saveSwitch(switcher.dialog.target, { apiKey });
          }
        }}
        onSaveCustom={(connection) => {
          void switcher.saveSwitch("custom", connection);
        }}
        onSwitchToHosted={() => {
          void switcher.saveSwitch("hosted");
        }}
      />
    </>
  );
}
