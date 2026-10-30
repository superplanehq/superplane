import type { useModelSourceSwitch } from "./useModelSourceSwitch";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { CodexConnectionSetupDialog } from "../../codex/CodexConnectionSetupDialog";
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
  return (
    <>
      <Dialog
        open={switcher.subscriptionDialog === "warn"}
        onOpenChange={(open) => {
          if (!open) switcher.setSubscriptionDialog(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Switch to Codex subscriptions?</DialogTitle>
            <DialogDescription>
              Connect your Codex account before this workspace switches. Intakes without a connected default owner will
              use your account.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => switcher.setSubscriptionDialog(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => switcher.setSubscriptionDialog("connect")}>
              Connect Codex
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <CodexConnectionSetupDialog
        organizationId={organizationId}
        factoryId={factoryId}
        open={switcher.subscriptionDialog === "connect"}
        onOpenChange={(open) => {
          if (!open) switcher.setSubscriptionDialog(null);
        }}
        onConnected={async () => {
          if (!(await switcher.saveSwitch("subscription")))
            throw new Error("Could not switch the model source. Try again.");
        }}
      />
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
