import { Icon } from "@/components/Icon";
import { ScheduledIntakeScheduleFields } from "./ScheduledIntakeScheduleFields";
import { useScheduledIntakeSchedule } from "./useScheduledIntakeSchedule";
import { useState } from "react";
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
import { Textarea } from "@/components/ui/textarea";
import { useCreateFactoryAutomation } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_SCHEDULED_INTAKE } from "@/lib/experimentalFeatures";
import { getApiErrorMessage } from "@/lib/errors";

const DEFAULT_PROMPT =
  "Inspect the repository and propose one small, concrete improvement. Verify the problem in the code. Explain what to change, why it matters, and how to verify the result. Do not propose work that is already complete.";

interface Props {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: (canvasId: string) => void;
}

export function ScheduledIntakeSetupDialog({ organizationId, factoryId, onClose, onCreated }: Props) {
  const enabled = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_SCHEDULED_INTAKE);
  const create = useCreateFactoryAutomation(organizationId, factoryId);
  const [name, setName] = useState("Scheduled intake");
  const schedule = useScheduledIntakeSchedule(organizationId, factoryId, enabled);
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [error, setError] = useState<string>();
  const valid = Boolean(name.trim() && prompt.trim() && schedule.ready);
  if (!enabled) return null;

  const submit = async () => {
    if (!valid || create.isPending) return;
    setError(undefined);
    try {
      const automation = await create.mutateAsync({
        name: name.trim(),
        templateId: "scheduled-intake",
        installParams: { ...schedule.params, prompt: prompt.trim() },
      });
      if (automation.id) onCreated(automation.id);
    } catch (cause) {
      setError(getApiErrorMessage(cause, "Could not create scheduled intake. Try again."));
    }
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !create.isPending) onClose();
      }}
    >
      <DialogContent className="sm:max-w-xl" showCloseButton={!create.isPending}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon name="alarm-clock" />
            Scheduled intake
          </DialogTitle>
          <DialogDescription>
            Inspect the workspace repository on a schedule and propose a task in Backlog.
          </DialogDescription>
        </DialogHeader>
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <fieldset disabled={create.isPending} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="scheduled-intake-name">Name</Label>
              <Input
                id="scheduled-intake-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </div>
            <ScheduledIntakeScheduleFields schedule={schedule} />
            <div className="space-y-2">
              <Label htmlFor="scheduled-intake-prompt">What should the agent look for?</Label>
              <Textarea
                id="scheduled-intake-prompt"
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                rows={6}
                required
              />
              <p className="text-sm text-muted-foreground">
                The agent uses the workspace repository and model settings. No task is created when it finds nothing
                useful.
              </p>
            </div>
          </fieldset>
          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button type="button" variant="outline" disabled={create.isPending} onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={!valid || create.isPending}>
              {create.isPending ? "Creating…" : "Create intake"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
