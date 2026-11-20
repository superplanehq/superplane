import type { FactoriesFactoryLine, FactoryLineStage } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { useUpdateFactoryLine } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { orderedLineBoardStages, type LineBoardStage } from "../lib/lineBoardStages";

export function AddLineStageDialog({
  organizationId,
  factoryId,
  line,
  open,
  onClose,
  onCreated,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  open: boolean;
  onClose: () => void;
  onCreated: (stage: LineBoardStage) => void;
}) {
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const stages = orderedLineBoardStages(line);
  const trimmedName = name.trim();
  const duplicate = stages.some((stage) => stage.name.toLocaleLowerCase() === trimmedName.toLocaleLowerCase());

  const close = () => {
    if (updateLine.isPending) {
      return;
    }
    setName("");
    setError("");
    onClose();
  };

  const create = async () => {
    if (!line.id || !trimmedName || duplicate) {
      return;
    }
    setError("");
    const temporaryKey = `new-${crypto.randomUUID()}`;
    const next = insertBeforeDone(stages, {
      id: "",
      key: temporaryKey,
      name: trimmedName,
      kind: "STAGE_KIND_CUSTOM",
    });
    try {
      const updated = await updateLine.mutateAsync({
        lineId: line.id,
        stageConfiguration: { stages: next },
      });
      const created = orderedLineBoardStages(updated).find(
        (stage) => stage.kind === "STAGE_KIND_CUSTOM" && !stages.some((existing) => existing.id === stage.id),
      );
      if (!created) {
        setError("SuperPlane could not find the new stage.");
        return;
      }
      setName("");
      onCreated(created);
    } catch (createError) {
      setError(getApiErrorMessage(createError, "SuperPlane could not add the stage."));
    }
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && close()}>
      <DialogContent
        className="sm:max-w-md"
        showCloseButton={!updateLine.isPending}
        data-testid="add-line-stage-dialog"
      >
        <DialogHeader>
          <DialogTitle>Add stage</DialogTitle>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="line-stage-name">Name</Label>
          <Input
            id="line-stage-name"
            value={name}
            maxLength={64}
            autoFocus
            onChange={(event) => {
              setName(event.target.value);
              setError("");
            }}
          />
          {duplicate ? <p className="text-sm text-destructive">Stage names must be unique.</p> : null}
          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={close} disabled={updateLine.isPending}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
            onClick={() => void create()}
            disabled={!trimmedName || duplicate}
            loading={updateLine.isPending}
            loadingText="Adding..."
          >
            Add stage
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function insertBeforeDone(stages: LineBoardStage[], stage: FactoryLineStage): FactoryLineStage[] {
  const doneIndex = stages.findIndex((entry) => entry.kind === "STAGE_KIND_DONE");
  const index = doneIndex < 0 ? stages.length : doneIndex;
  return [...stages.slice(0, index), stage, ...stages.slice(index)];
}
