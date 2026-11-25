import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Webhook } from "lucide-react";
import { useState } from "react";

import { CustomAutomationWebhookFields } from "./CustomAutomationWebhookFields";
import { IntakeSetupDefaultOwner } from "./IntakeDefaultOwnerField";
import { WebhookAuthenticationField, WebhookSecretDialog } from "./settings/FactorySettingsWebhooksPage";
import { useWebhookIntakeSetup, type WebhookIntakeMode, type WebhookIntakeSetupModel } from "./useWebhookIntakeSetup";
import { WebhookIntakeFieldPathInputs } from "./WebhookIntakeFieldPathInputs";
import { webhookIntakeExampleBody } from "./webhookIntakeFieldPaths";
import { WEBHOOK_INTAKE_SETUP_COPY } from "./webhookIntakeSetupCopy";

interface WebhookIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  onClose: () => void;
  onCreated: () => void;
}

export function WebhookIntakeSetupDialog(props: WebhookIntakeSetupDialogProps) {
  const setup = useWebhookIntakeSetup(props.organizationId, props.factoryId);
  const [step, setStep] = useState<"webhook" | "fields">("webhook");
  const fieldsStep = step === "fields";

  if (setup.reveal) {
    return (
      <WebhookSecretDialog
        reveal={setup.reveal}
        payload={webhookIntakeExampleBody(setup.paths)}
        onClose={props.onCreated}
      />
    );
  }

  const submit = () => {
    if (!fieldsStep) {
      if (setup.webhookReady) setStep("fields");
      return;
    }
    if (!setup.canFinish) return;
    void setup.finish().then((result) => {
      if (result === "created") props.onCreated();
    });
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !setup.pending) props.onClose();
      }}
    >
      <DialogContent
        className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl"
        showCloseButton={!setup.pending}
        data-testid="webhook-intake-setup"
      >
        <DialogHeader>
          <p className="text-xs text-muted-foreground" data-testid="webhook-intake-setup-step">
            {fieldsStep ? WEBHOOK_INTAKE_SETUP_COPY.stepFields : WEBHOOK_INTAKE_SETUP_COPY.stepWebhook}
          </p>
          <DialogTitle className="flex items-center gap-2">
            <Webhook className="size-4" aria-hidden />
            {fieldsStep ? WEBHOOK_INTAKE_SETUP_COPY.fieldsTitle : WEBHOOK_INTAKE_SETUP_COPY.title}
          </DialogTitle>
          <DialogDescription>
            {fieldsStep ? WEBHOOK_INTAKE_SETUP_COPY.fieldsHelper : WEBHOOK_INTAKE_SETUP_COPY.description}
          </DialogDescription>
        </DialogHeader>
        <form
          className="min-w-0 space-y-6"
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <fieldset disabled={setup.pending} className="min-w-0 space-y-6">
            {fieldsStep ? (
              <>
                <WebhookIntakeFieldPathInputs
                  idPrefix="webhook-intake-setup"
                  paths={setup.paths}
                  disabled={setup.pending}
                  onChange={setup.setPaths}
                />
                <IntakeSetupDefaultOwner owner={setup.owner} disabled={setup.pending} />
              </>
            ) : (
              <WebhookSourceFields setup={setup} {...props} />
            )}
          </fieldset>
          {setup.error ? (
            <p role="alert" className="text-sm text-destructive">
              {setup.error}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={setup.pending}
              onClick={fieldsStep ? () => setStep("webhook") : props.onClose}
            >
              {fieldsStep ? WEBHOOK_INTAKE_SETUP_COPY.back : WEBHOOK_INTAKE_SETUP_COPY.cancel}
            </Button>
            {fieldsStep ? (
              <Button type="submit" disabled={!setup.canFinish} data-testid="webhook-intake-setup-finish">
                {setup.pending ? WEBHOOK_INTAKE_SETUP_COPY.creating : WEBHOOK_INTAKE_SETUP_COPY.create}
              </Button>
            ) : (
              <Button type="submit" disabled={!setup.webhookReady} data-testid="webhook-intake-setup-continue">
                {WEBHOOK_INTAKE_SETUP_COPY.continue}
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

const MODES: Array<{ id: WebhookIntakeMode; label: string; helper: string }> = [
  { id: "new", label: WEBHOOK_INTAKE_SETUP_COPY.modeNew, helper: WEBHOOK_INTAKE_SETUP_COPY.modeNewHelper },
  {
    id: "existing",
    label: WEBHOOK_INTAKE_SETUP_COPY.modeExisting,
    helper: WEBHOOK_INTAKE_SETUP_COPY.modeExistingHelper,
  },
];

function WebhookSourceFields({
  setup,
  organizationId,
  factoryId,
  factoryKey,
}: { setup: WebhookIntakeSetupModel } & WebhookIntakeSetupDialogProps) {
  const locked = Boolean(setup.createdWebhook);
  return (
    <div className="space-y-4">
      <div
        className="grid gap-2 sm:grid-cols-2"
        role="radiogroup"
        aria-label={WEBHOOK_INTAKE_SETUP_COPY.webhookSection}
      >
        {MODES.map((mode) => {
          const selected = setup.mode === mode.id;
          return (
            <button
              key={mode.id}
              type="button"
              role="radio"
              aria-checked={selected}
              disabled={locked && !selected}
              onClick={() => setup.setMode(mode.id)}
              data-testid={`webhook-intake-mode-${mode.id}`}
              className={cn(
                "flex items-start gap-2.5 rounded-lg border px-3 py-2.5 text-left transition-colors disabled:opacity-50",
                selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
              )}
            >
              <span
                className={cn(
                  "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border",
                  selected ? "border-foreground" : "border-muted-foreground/50",
                )}
                aria-hidden
              >
                {selected ? <span className="size-2 rounded-full bg-foreground" /> : null}
              </span>
              <span className="min-w-0">
                <span className="block text-[13px] font-medium">{mode.label}</span>
                <span className="block text-xs text-muted-foreground">{mode.helper}</span>
              </span>
            </button>
          );
        })}
      </div>
      {setup.mode === "new" ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="webhook-intake-name">{WEBHOOK_INTAKE_SETUP_COPY.nameLabel}</Label>
            <Input
              id="webhook-intake-name"
              value={setup.name}
              maxLength={100}
              disabled={locked}
              onChange={(event) => setup.setName(event.target.value)}
              data-testid="webhook-intake-name"
            />
          </div>
          <WebhookAuthenticationField
            value={setup.authentication}
            onChange={setup.setAuthentication}
            disabled={locked}
          />
        </div>
      ) : (
        <CustomAutomationWebhookFields
          idPrefix="webhook-intake"
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          webhookId={setup.webhookId}
          usedWebhookIds={[]}
          onChange={setup.setWebhookId}
          helper={WEBHOOK_INTAKE_SETUP_COPY.existingHelper}
        />
      )}
    </div>
  );
}
