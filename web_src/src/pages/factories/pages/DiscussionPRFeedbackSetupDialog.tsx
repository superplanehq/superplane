import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";

import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { ReviewBotPicker } from "./ReviewBotPicker";
import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { PR_FEEDBACK_SETTINGS_COPY, type PRFeedbackSource } from "./prFeedbackSettingsModel";
import {
  useDiscussionPRFeedbackSetup,
  type DiscussionBotMode,
  type DiscussionPRFeedbackSetupModel,
} from "./useDiscussionPRFeedbackSetup";

interface DiscussionPRFeedbackSetupDialogProps {
  organizationId: string;
  factoryId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}

export function DiscussionPRFeedbackSetupDialog(props: DiscussionPRFeedbackSetupDialogProps) {
  const setup = useDiscussionPRFeedbackSetup(props.organizationId, props.factoryId, props.repository);
  const canLeaveStep = setup.step === "bots";

  return (
    <div className="space-y-6" data-testid="discussion-pr-feedback-setup">
      <SetupHeader
        step={setup.step}
        onBack={() => {
          if (canLeaveStep) {
            setup.setStep("mention");
            return;
          }
          props.onClose();
        }}
      />
      <div>
        {setup.step === "mention" ? <MentionStep setup={setup} /> : <BotsStep setup={setup} />}
        {setup.error ? (
          <p className="workspace-body-text mt-4 text-destructive" role="alert">
            {setup.error}
          </p>
        ) : null}
      </div>
      <SetupFooter setup={setup} source={props.source} onClose={props.onClose} onCreated={props.onCreated} />
    </div>
  );
}

function SetupHeader({ step, onBack }: { step: "mention" | "bots"; onBack: () => void }) {
  const stepTitle =
    step === "mention"
      ? PR_FEEDBACK_SETTINGS_COPY.wizardStepHumanComments
      : PR_FEEDBACK_SETTINGS_COPY.wizardStepAIComments;
  const stepIntro = step === "bots" ? PR_FEEDBACK_SETTINGS_COPY.wizardBotsIntro : undefined;

  return (
    <header className="text-left">
      <button
        type="button"
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
        onClick={onBack}
        data-testid="discussion-setup-back"
      >
        <ArrowLeft className="size-4 shrink-0" aria-hidden />
        <span>
          {step === "bots" ? PR_FEEDBACK_SETTINGS_COPY.wizardBack : PR_FEEDBACK_SETTINGS_COPY.wizardBackToBoard}
        </span>
      </button>
      <h1 className={factoryPageTitleClassName}>{stepTitle}</h1>
      {stepIntro ? <p className="workspace-body-text mt-2 text-muted-foreground">{stepIntro}</p> : null}
    </header>
  );
}

function MentionStep({ setup }: { setup: DiscussionPRFeedbackSetupModel }) {
  return (
    <section
      className="flex flex-col gap-2"
      role="radiogroup"
      aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardStepHumanComments}
    >
      <IntakeSettingsRadioOption
        name="discussion-setup-mention"
        value="require"
        checked={setup.mentionRequired}
        title={PR_FEEDBACK_SETTINGS_COPY.wizardMentionRequireOption}
        helper={PR_FEEDBACK_SETTINGS_COPY.wizardMentionRequireHelper}
        onChange={() => setup.setMentionRequired(true)}
      />
      <IntakeSettingsRadioOption
        name="discussion-setup-mention"
        value="any"
        checked={!setup.mentionRequired}
        title={PR_FEEDBACK_SETTINGS_COPY.wizardMentionAnyOption}
        helper={PR_FEEDBACK_SETTINGS_COPY.wizardMentionAnyHelper}
        onChange={() => setup.setMentionRequired(false)}
      />
    </section>
  );
}

const BOT_MODE_OPTIONS: Array<{ value: DiscussionBotMode; title: string; helper: string }> = [
  {
    value: "ignore",
    title: PR_FEEDBACK_SETTINGS_COPY.wizardBotIgnoreOption,
    helper: PR_FEEDBACK_SETTINGS_COPY.wizardBotIgnoreHelper,
  },
  {
    value: "address",
    title: PR_FEEDBACK_SETTINGS_COPY.wizardBotAddressOption,
    helper: PR_FEEDBACK_SETTINGS_COPY.wizardBotAddressHelper,
  },
];

function BotsStep({ setup }: { setup: DiscussionPRFeedbackSetupModel }) {
  return (
    <div className="space-y-5">
      <section
        className="flex flex-col gap-2"
        role="radiogroup"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardStepAIComments}
      >
        {BOT_MODE_OPTIONS.map((option) => (
          <IntakeSettingsRadioOption
            key={option.value}
            name="discussion-setup-bot-mode"
            value={option.value}
            checked={setup.botMode === option.value}
            title={option.title}
            helper={option.helper}
            onChange={() => setup.setBotMode(option.value)}
          />
        ))}
      </section>
      {setup.botMode === "address" ? (
        <ReviewBotPicker
          selected={setup.allowedBots}
          catalog={setup.catalog}
          loading={setup.catalogLoading}
          loadError={setup.catalogQuery.isError}
          onToggle={setup.toggleBot}
          onAdd={setup.addBot}
        />
      ) : null}
    </div>
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
    <footer className="flex items-center justify-between gap-3 pt-2">
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
