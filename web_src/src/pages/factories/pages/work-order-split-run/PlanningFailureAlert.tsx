import { useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { TaskRefinementModelSelect } from "../../TaskRefinementModelSelect";

const LEGACY_MODEL_UNAVAILABLE =
  /^model "([^"]+)": the selected model is unavailable for this task owner; select another model or clear the override$/;

function planningFailureMessage(failure: string): string {
  const model = LEGACY_MODEL_UNAVAILABLE.exec(failure)?.[1];
  return model ? `Model "${model}" is not available for the current task owner.` : failure;
}

function isModelUnavailableFailure(failure: string): boolean {
  return /\bmodel\b.*\b(?:unavailable|not available)\b/i.test(failure);
}

export function PlanningFailureAlert({
  failure,
  organizationId,
  canEditModel = false,
  hasSession = false,
}: {
  failure?: string;
  organizationId?: string;
  canEditModel?: boolean;
  hasSession?: boolean;
}) {
  const [modelError, setModelError] = useState("");
  useEffect(() => setModelError(""), [failure]);
  if (!failure) {
    return null;
  }
  return (
    <Alert variant="destructive" aria-label="Planning failed" className="mt-3">
      <AlertTitle>Planning failed</AlertTitle>
      <AlertDescription className="whitespace-pre-wrap break-words">
        <p>{planningFailureMessage(failure)}</p>
        {hasSession ? (
          <p>Select a model in the task header, then send a new message to continue.</p>
        ) : isModelUnavailableFailure(failure) ? (
          <p>Change the model in Planning settings, or select another model for this task below.</p>
        ) : null}
        {!hasSession ? (
          <TaskRefinementModelSelect
            organizationId={organizationId}
            canEdit={canEditModel}
            allowReselect
            onSaveStart={() => setModelError("")}
            onError={setModelError}
          />
        ) : null}
        {modelError ? <p className="text-destructive">{modelError}</p> : null}
      </AlertDescription>
    </Alert>
  );
}
