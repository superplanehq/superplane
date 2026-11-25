import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";

import { CustomAutomationWebhookFields } from "./CustomAutomationWebhookFields";
import { IntakeSetupDefaultOwner } from "./IntakeDefaultOwnerField";
import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { WebhookAuthenticationField, WebhookSecretDialog } from "./settings/FactorySettingsWebhooksPage";
import { useWebhookIntakeSetup, type WebhookIntakeMode, type WebhookIntakeSetupModel } from "./useWebhookIntakeSetup";
import { WebhookIntakeFieldPathInputs } from "./WebhookIntakeFieldPathInputs";
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
  const webhookStep = setup.step === "connection";

  return (
    <>
      <IntakeSetupWizard
        testId="webhook-intake-setup"
        integrationName="webhook"
        step={setup.step}
        title={webhookStep ? WEBHOOK_INTAKE_SETUP_COPY.wizardStepWebhook : WEBHOOK_INTAKE_SETUP_COPY.wizardStepFields}
        helper={
          webhookStep
            ? WEBHOOK_INTAKE_SETUP_COPY.wizardStepWebhookHelper
            : WEBHOOK_INTAKE_SETUP_COPY.wizardStepFieldsHelper
        }
        resourceStepLabel="Map request fields"
        resourceStepCaption="Awaiting field mapping"
        footer={<SetupFooter setup={setup} onCreated={props.onCreated} />}
        onBack={() => {
          if (setup.step === "project") {
            setup.setStep("connection");
            return;
          }
          props.onClose();
        }}
      >
        <div>
          {webhookStep ? (
            <WebhookStep setup={setup} {...props} />
          ) : (
            <WebhookIntakeFieldPathInputs
              idPrefix="webhook-intake-setup"
              paths={setup.paths}
              disabled={setup.pending}
              onChange={setup.setPaths}
            />
          )}
          {setup.error ? (
            <p className="workspace-body-text mt-4 text-destructive" role="alert">
              {setup.error}
            </p>
          ) : null}
        </div>
        {webhookStep ? null : <IntakeSetupDefaultOwner owner={setup.owner} disabled={setup.pending} />}
      </IntakeSetupWizard>
      <WebhookSecretDialog reveal={setup.reveal} onClose={props.onCreated} />
    </>
  );
}

const MODES: Array<{ id: WebhookIntakeMode; label: string }> = [
  { id: "new", label: WEBHOOK_INTAKE_SETUP_COPY.modeNew },
  { id: "existing", label: WEBHOOK_INTAKE_SETUP_COPY.modeExisting },
];

function WebhookStep({
  setup,
  organizationId,
  factoryId,
  factoryKey,
}: { setup: WebhookIntakeSetupModel } & WebhookIntakeSetupDialogProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-2" role="radiogroup" aria-label="Webhook">
        {MODES.map((mode) => {
          const selected = setup.mode === mode.id;
          return (
            <Button
              key={mode.id}
              type="button"
              variant="ghost"
              role="radio"
              aria-checked={selected}
              onClick={() => setup.setMode(mode.id)}
              data-testid={`webhook-intake-mode-${mode.id}`}
              className={cn(
                "h-auto justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
                selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
              )}
            >
              <span className="min-w-0 flex-1 truncate">{mode.label}</span>
              {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
            </Button>
          );
        })}
      </div>
      {setup.mode === "new" ? (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="webhook-intake-name">{WEBHOOK_INTAKE_SETUP_COPY.nameLabel}</Label>
            <Input
              id="webhook-intake-name"
              value={setup.name}
              maxLength={100}
              disabled={Boolean(setup.createdWebhook)}
              onChange={(event) => setup.setName(event.target.value)}
              data-testid="webhook-intake-name"
            />
          </div>
          <WebhookAuthenticationField value={setup.authentication} onChange={setup.setAuthentication} />
          <p className="text-xs text-muted-foreground">{WEBHOOK_INTAKE_SETUP_COPY.newWebhookHelper}</p>
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

function SetupFooter({ setup, onCreated }: { setup: WebhookIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <Button
        type="button"
        className="w-full"
        disabled={!setup.canContinue}
        onClick={() => setup.setStep("project")}
        data-testid="webhook-intake-setup-continue"
      >
        {WEBHOOK_INTAKE_SETUP_COPY.wizardContinue}
      </Button>
    );
  }

  return (
    <Button
      type="button"
      className="w-full"
      disabled={!setup.canFinish}
      onClick={() => {
        void setup.finish().then((result) => {
          if (result === "created") onCreated();
        });
      }}
      data-testid="webhook-intake-setup-finish"
    >
      {setup.pending ? WEBHOOK_INTAKE_SETUP_COPY.wizardFinishing : WEBHOOK_INTAKE_SETUP_COPY.wizardFinish}
    </Button>
  );
}
