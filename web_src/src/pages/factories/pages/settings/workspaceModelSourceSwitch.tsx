import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import { cn } from "@/lib/utils";
import { CodexConnectionSetupContent } from "../../codex/CodexConnectionSetupDialog";
import { useCodexConnectionSetup } from "../../codex/useCodexConnectionSetup";
import { IntakeDefaultOwnerField } from "../IntakeDefaultOwnerField";
import {
  type LLMModelsSwitchDialog,
  type LLMModelsSwitchProvider,
  SwitchDialogContent,
} from "./FactorySettingsLLMModelsSwitchPreview";
import { renderModelSourceSwitchHeader } from "./modelSourceSwitchHeader";
import { byokProviderProductName } from "./organizationLLMModelsCopy";
import type { useWorkspaceModelSettings } from "./useWorkspaceModelSettings";
import { WorkspaceModelSourceSelector } from "./WorkspaceModelSourceSelector";

type Settings = ReturnType<typeof useWorkspaceModelSettings>;

export function WorkspaceModelSourceDialog({ settings }: { settings: Settings }) {
  const { switcher } = settings;
  const dialog = switcher.dialog;
  const choosing = dialog?.step === "choose";
  const setup = useWorkspaceCodexSetup(settings);
  const closeDialog = async () => {
    if (switcher.saving || setup.busy) return;
    if (dialog?.step === "connect" && !(await setup.cancelSetup())) return;
    switcher.closeDialog();
  };
  return (
    <Dialog
      open={dialog !== null}
      onOpenChange={(open) => {
        if (!open) void closeDialog();
      }}
    >
      <DialogContent
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          settings.changeSourceButtonRef.current?.focus();
        }}
        size="large"
        className={cn("max-h-[90vh] w-[calc(100%-2rem)] overflow-y-auto", choosing ? "max-w-3xl" : "max-w-lg")}
      >
        {renderSourceDialogHeader(settings, setup)}
        {dialog?.step !== "subscription" && dialog?.step !== "connect" && (
          <SourceDialogBackButton settings={settings} setup={setup} />
        )}
        {renderSourceDialogBody(settings, setup)}
      </DialogContent>
    </Dialog>
  );
}

type CodexSetup = ReturnType<typeof useCodexConnectionSetup>;

function SourceDialogBackButton({ settings, setup }: { settings: Settings; setup: CodexSetup }) {
  const { switcher } = settings;
  const dialog = switcher.dialog;
  if (!dialog || dialog.step === "choose") return null;
  const compact = dialog.step === "subscription" || dialog.step === "connect";
  const label = dialog.step === "connect" ? "Back to owner selection" : "Back to sources";
  return (
    <Button
      type="button"
      variant="ghost"
      size={compact ? "icon" : "sm"}
      className={compact ? "-ml-2 size-8 shrink-0" : "w-fit px-0"}
      title={compact ? label : undefined}
      disabled={switcher.saving || setup.busy}
      onClick={() => {
        if (dialog.step === "connect") {
          void setup.closeSetup();
          return;
        }
        switcher.setDialog({ step: "choose" });
      }}
    >
      <ArrowLeft className="size-4" aria-hidden />
      <span className={compact ? "sr-only" : undefined}>{label}</span>
    </Button>
  );
}

function renderSourceDialogBody(settings: Settings, setup: CodexSetup) {
  const { dialog } = settings.switcher;
  if (!dialog) return null;
  switch (dialog.step) {
    case "choose":
      return <WorkspaceModelSourceSelector settings={settings} />;
    case "subscription":
      return <SubscriptionOwnerStep settings={settings} />;
    case "connect":
      return <CodexConnectionSetupContent setup={setup} />;
    case "warn":
    case "key":
      return <ProviderSwitchStep settings={settings} dialog={dialog} />;
  }
}

function ProviderSwitchStep({ settings, dialog }: { settings: Settings; dialog: NonNullable<LLMModelsSwitchDialog> }) {
  const { switcher, source, currentProvider } = settings;
  return (
    <SwitchDialogContent
      source={source === "hosted" || source === "subscription" ? source : (currentProvider ?? "anthropic")}
      dialog={dialog}
      saving={switcher.saving}
      providerSelector={dialog.target !== "hosted" ? <ProviderSelector settings={settings} /> : undefined}
      onCancel={switcher.closeDialog}
      onContinueToKey={() => {
        if (dialog.target === "hosted") return;
        if (settings.connected.includes(dialog.target)) {
          void switcher.saveSwitch(dialog.target);
          return;
        }
        switcher.setDialog({ target: dialog.target, step: "key" });
      }}
      onBackToWarning={() => switcher.setDialog({ target: dialog.target, step: "warn" })}
      onSaveKey={(apiKey) => {
        if (dialog.target !== "hosted") void switcher.saveSwitch(dialog.target, { apiKey });
      }}
      onSaveCustom={(connection) => void switcher.saveSwitch("custom", connection)}
      onSwitchToHosted={() => void switcher.saveSwitch("hosted")}
    />
  );
}

function renderSourceDialogHeader(settings: Settings, setup: CodexSetup) {
  const { dialog } = settings.switcher;
  if (!dialog) return null;
  if (dialog.step === "warn" || dialog.step === "key") {
    return renderModelSourceSwitchHeader(
      settings.source === "own-key" ? (settings.currentProvider ?? "anthropic") : settings.source,
      dialog,
    );
  }
  if (dialog.step === "choose") {
    return (
      <DialogHeader>
        <DialogTitle>Change model source</DialogTitle>
        <DialogDescription>Choose how agents in this workspace access models.</DialogDescription>
      </DialogHeader>
    );
  }
  if (dialog.step === "connect") {
    return (
      <DialogHeader className="text-left">
        <div className="flex items-center gap-2 pr-4">
          <SourceDialogBackButton settings={settings} setup={setup} />
          <DialogTitle>Connect Codex</DialogTitle>
        </div>
        <DialogDescription>Connect once to use Codex across this organization’s workspaces.</DialogDescription>
      </DialogHeader>
    );
  }
  return (
    <DialogHeader className="text-left">
      <div className="flex items-center gap-2 pr-4">
        <SourceDialogBackButton settings={settings} setup={setup} />
        <DialogTitle>
          {settings.source === "subscription" ? "Change default task owner" : "Switch to Codex"}
        </DialogTitle>
      </div>
    </DialogHeader>
  );
}

function useWorkspaceCodexSetup(settings: Settings) {
  const { currentUserId } = usePermissions();
  const backToOwner = () => settings.switcher.setDialog({ step: "subscription" });
  return useCodexConnectionSetup({
    organizationId: settings.organizationId,
    factoryId: settings.factoryId,
    open: settings.switcher.dialog?.step === "connect",
    onClose: backToOwner,
    onConnected: () => {
      if (currentUserId) settings.switcher.setDefaultOwnerId(currentUserId);
      backToOwner();
    },
  });
}

function SubscriptionOwnerStep({ settings }: { settings: Settings }) {
  const { organizationId, factoryId, switcher } = settings;
  const eligibility = useFactoryOwnerEligibility(organizationId, factoryId, undefined, true);
  const changingOwner = settings.source === "subscription";
  const selectedEligible =
    eligibility.data?.members?.some((member) => member.userId === switcher.defaultOwnerId && member.eligible) === true;
  const noConnectedMembers =
    eligibility.isSuccess && !eligibility.data.members?.some((member) => member.userId && member.eligible);
  return (
    <>
      {noConnectedMembers ? (
        <section className="flex flex-col gap-3" aria-label="Default task owner">
          <p className="text-sm font-medium">Default task owner (required)</p>
          <div className="flex flex-col items-start gap-3 rounded-md border border-border p-4">
            <p className="text-sm text-muted-foreground">No members have ChatGPT connected.</p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={switcher.saving}
              onClick={() => switcher.setDialog({ step: "connect" })}
            >
              Connect Codex
            </Button>
          </div>
        </section>
      ) : (
        <IntakeDefaultOwnerField
          organizationId={organizationId}
          factoryId={factoryId}
          required
          prospectiveSubscription
          description="Select a member with ChatGPT connected."
          value={switcher.defaultOwnerId}
          onChange={switcher.setDefaultOwnerId}
          disabled={switcher.saving}
          onConnectCodex={() => switcher.setDialog({ step: "connect" })}
        />
      )}
      <div className="space-y-1 border-t border-border pt-4 text-sm">
        <p className="font-medium">Applies to all intakes</p>
        <p className="text-muted-foreground">
          Replaces every default owner, including scheduled intakes and intakes with another owner.
        </p>
      </div>
      {switcher.subscriptionError ? <p role="alert">{switcher.subscriptionError}</p> : null}
      <DialogFooter>
        <Button type="button" variant="outline" disabled={switcher.saving} onClick={switcher.closeDialog}>
          Cancel
        </Button>
        <Button
          type="button"
          disabled={!selectedEligible || switcher.saving || eligibility.isFetching}
          onClick={() => void switcher.saveSwitch("subscription")}
        >
          {changingOwner ? "Save default owner" : "Switch to Codex"}
        </Button>
      </DialogFooter>
    </>
  );
}

function ProviderSelector({ settings }: { settings: Settings }) {
  const { switcher } = settings;
  const dialog = switcher.dialog;
  const target = dialog && "target" in dialog ? dialog.target : "";
  return (
    <Select
      value={target}
      onValueChange={(provider) => switcher.setDialog({ target: provider as LLMModelsSwitchProvider, step: "warn" })}
      disabled={switcher.saving}
    >
      <SelectTrigger aria-label="New API key provider">
        <SelectValue placeholder="Select a provider" />
      </SelectTrigger>
      <SelectContent>
        {settings.providers
          .filter((provider) => settings.canCreateIntegration || settings.connected.includes(provider))
          .map((provider) => (
            <SelectItem key={provider} value={provider}>
              {byokProviderProductName(provider)}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  );
}
