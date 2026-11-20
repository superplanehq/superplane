import type { FactoriesFactoryLine } from "@/api-client";
import { useUpdateFactoryLine } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { useState } from "react";

import { orderedLineBoardStages, type LineBoardStage } from "../lib/lineBoardStages";

export function DeleteLineStageDialog({
  organizationId,
  factoryId,
  line,
  stage,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  stage: LineBoardStage | null;
  onClose: () => void;
}) {
  const [error, setError] = useState("");
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);

  const remove = async () => {
    if (!line.id || !stage) {
      return;
    }
    setError("");
    try {
      await updateLine.mutateAsync({
        lineId: line.id,
        stageConfiguration: { stages: orderedLineBoardStages(line).filter((entry) => entry.id !== stage.id) },
      });
      onClose();
    } catch (removeError) {
      setError(getApiErrorMessage(removeError, "SuperPlane could not delete the stage."));
    }
  };

  return (
    <AlertDialog
      open={Boolean(stage)}
      onOpenChange={(open) => {
        if (!open && !updateLine.isPending) {
          setError("");
          onClose();
        }
      }}
    >
      <AlertDialogContent data-testid="delete-line-stage-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {stage?.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            This removes the stage from this line. Its automations stay in the workspace.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error ? (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={updateLine.isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            disabled={updateLine.isPending}
            onClick={(event) => {
              event.preventDefault();
              void remove();
            }}
          >
            Delete stage
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
