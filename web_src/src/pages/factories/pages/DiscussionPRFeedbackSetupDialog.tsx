import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft, Check } from "lucide-react";

import { ReviewBotPicker } from "./ReviewBotPicker";
import { PR_FEEDBACK_SETTINGS_COPY, type PRFeedbackSource } from "./prFeedbackSettingsModel";
import { useDiscussionPRFeedbackSetup, type DiscussionPRFeedbackSetupModel } from "./useDiscussionPRFeedbackSetup";

interface DiscussionPRFeedbackSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
  /** Dialog overlay (default) or inline card for a dedicated setup page. */
  presentation?: "dialog" | "page";
}

export function DiscussionPRFeedbackSetupDialog(props: DiscussionPRFeedbackSetupDialogProps) {
  const presentation = props.presentation ?? "dialog";
  const setup = useDiscussionPRFeedbackSetup(
    props.organizationId,
    props.factoryId,
    props.repository,
    props.open || presentation === "page",
  );
  const body = (
    <DiscussionSetupBody setup={setup} source={props.source} onClose={props.onClose} onCreated={props.onCreated} />
  );

  if (presentation === "page") {
    return (
      <div
        className="grid h-[min(36rem,80vh)] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden rounded-lg border border-border bg-background"
        data-testid="discussion-pr-feedback-setup"
      >
        {body}
      </div>
    );
  }

  return (
    <Dialog open={props.open} onOpenChange={(next) => !next && props.onClose()}>
      <DialogContent
        className="grid h-[min(36rem,80vh)] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0 sm:max-w-xl"
        showCloseButton
        data-testid="discussion-pr-feedback-setup"
      >
        {body}
      </DialogContent>
    </Dialog>
  );
}

function DiscussionSetupBody({
  setup,
  source,
  onClose,
  onCreated,
}: {
  setup: DiscussionPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  return (
    <>
      <SetupHeader step={setup.step} onBack={() => setup.setStep("mention")} />
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5">
        {setup.step === "mention" ? <MentionStep setup={setup} /> : <BotsStep setup={setup} />}
        {setup.error ? (
          <p className="workspace-body-text mt-4 text-destructive" role="alert">
            {setup.error}
          </p>
        ) : null}
      </div>
      <SetupFooter setup={setup} source={source} onClose={onClose} onCreated={onCreated} />
    </>
  );
}

function SetupHeader({ step, onBack }: { step: "mention" | "bots"; onBack: () => void }) {
  return (
    <header className="shrink-0 border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2.5">
        {step === "bots" ? (
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={onBack}
            aria-label="Go back"
          >
            <ArrowLeft className="size-4" />
          </button>
        ) : null}
        <IntegrationIcon integrationName="github" className="size-5 shrink-0" size={20} />
        <h2 className="min-w-0 text-[15px] font-semibold leading-5">
          {PR_FEEDBACK_SETTINGS_COPY.wizardCommentsTitle}
        </h2>
      </div>
      <p className="sr-only">
        {step === "bots"
          ? PR_FEEDBACK_SETTINGS_COPY.wizardBotsHelper
          : PR_FEEDBACK_SETTINGS_COPY.wizardMentionDescription}
      </p>
    </header>
  );
}

function MentionStep({ setup }: { setup: DiscussionPRFeedbackSetupModel }) {
  return (
    <section>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-100">
        {PR_FEEDBACK_SETTINGS_COPY.wizardMentionDescription}
      </p>
      <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardMentionHelper}</p>
      <div className="mt-2 rounded-lg border border-border" role="listbox" aria-multiselectable="false">
        <button
          type="button"
          role="option"
          aria-selected={setup.mentionRequired}
          onClick={() => setup.setMentionRequired(!setup.mentionRequired)}
          className={cn(
            "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
            setup.mentionRequired ? "bg-accent/50" : "hover:bg-accent/30",
          )}
          data-testid="discussion-setup-mention"
        >
          <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
            {PR_FEEDBACK_SETTINGS_COPY.wizardMentionOption}
          </span>
          <span className="flex size-3.5 shrink-0 items-center justify-center">
            {setup.mentionRequired ? (
              <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden />
            ) : null}
          </span>
        </button>
      </div>
    </section>
  );
}

function BotsStep({ setup }: { setup: DiscussionPRFeedbackSetupModel }) {
  return (
    <ReviewBotPicker
      selected={setup.allowedBots}
      catalog={setup.catalog}
      loading={setup.catalogLoading}
      loadError={setup.catalogQuery.isError}
      onToggle={setup.toggleBot}
      onAdd={setup.addBot}
    />
  );
}

function SetupFooter({
  setup,
  source,
  onClose,
  onCreated,
}: {
  setup: DiscussionPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  return (
    <footer className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
      <span className="text-[12px] text-muted-foreground">
        {setup.step === "mention"
          ? PR_FEEDBACK_SETTINGS_COPY.wizardStepMention
          : PR_FEEDBACK_SETTINGS_COPY.wizardStepBots}
      </span>
      <div className="flex items-center gap-2">
        {setup.step === "mention" ? (
          <Button type="button" onClick={() => setup.setStep("bots")} data-testid="discussion-setup-continue">
            {PR_FEEDBACK_SETTINGS_COPY.wizardContinue}
          </Button>
        ) : (
          <Button
            type="button"
            disabled={setup.createHandler.isPending}
            onClick={() => {
              void setup.finish(source).then((handler) => {
                if (handler?.id) {
                  onCreated(handler.id);
                  onClose();
                }
              });
            }}
            data-testid="discussion-setup-finish"
          >
            {setup.createHandler.isPending
              ? PR_FEEDBACK_SETTINGS_COPY.wizardFinishing
              : PR_FEEDBACK_SETTINGS_COPY.wizardFinish}
          </Button>
        )}
      </div>
    </footer>
  );
}
