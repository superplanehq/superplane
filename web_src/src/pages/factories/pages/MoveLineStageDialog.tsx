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

import type { LineBoardStage } from "../lib/lineBoardStages";

export function MoveLineStageDialog({
  organizationId,
  factoryId,
  line,
  stages,
  movedStage,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  stages: LineBoardStage[] | null;
  movedStage: LineBoardStage | null;
  onClose: () => void;
}) {
  const [error, setError] = useState("");
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const movedIndex = stages?.findIndex((stage) => stage.id === movedStage?.id) ?? -1;
  const previousStage = movedIndex > 0 ? stages?.[movedIndex - 1] : undefined;

  const move = async () => {
    if (!line.id || !stages || !movedStage) return;
    setError("");
    try {
      await updateLine.mutateAsync({ lineId: line.id, stageConfiguration: { stages } });
      onClose();
    } catch (moveError) {
      setError(getApiErrorMessage(moveError, "SuperPlane could not move the stage."));
    }
  };

  return (
    <AlertDialog
      open={Boolean(stages && movedStage)}
      onOpenChange={(open) => {
        if (!open && !updateLine.isPending) {
          setError("");
          onClose();
        }
      }}
    >
      <AlertDialogContent data-testid="move-line-stage-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Move {movedStage?.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            Place this stage after {previousStage?.name}. This changes the stage order for everyone who uses this line.
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
            disabled={updateLine.isPending}
            onClick={(event) => {
              event.preventDefault();
              void move();
            }}
          >
            Move stage
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
