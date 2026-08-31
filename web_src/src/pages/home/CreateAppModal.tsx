import { useEffect, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { getApiErrorMessage } from "@/lib/errors";

const MAX_APP_NAME_LENGTH = 50;

interface CreateAppModalProps {
  open: boolean;
  isSaving: boolean;
  onClose: () => void;
  onCreate: (name: string) => Promise<void>;
}

export function CreateAppModal({ open, isSaving, onClose, onCreate }: CreateAppModalProps) {
  const [name, setName] = useState("");
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (!open) return;

    setName("");
    setNameError("");
  }, [open]);

  const handleClose = () => {
    if (isSaving) return;

    onClose();
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (isSaving) return;

    const appName = name.trim() || generateCanvasName();

    try {
      await onCreate(appName);
    } catch (error) {
      const errorMessage = getApiErrorMessage(error, "Failed to create app");
      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("An app with this name already exists");
      }
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) handleClose();
      }}
    >
      <DialogContent showCloseButton={false} className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create app</DialogTitle>
        </DialogHeader>

        <form onSubmit={(event) => void handleSubmit(event)}>
          <div className="space-y-2">
            <Label htmlFor="create-app-name-input">App name (optional)</Label>
            <Input
              id="create-app-name-input"
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                setNameError("");
              }}
              maxLength={MAX_APP_NAME_LENGTH}
              aria-invalid={Boolean(nameError)}
              aria-describedby={`create-app-name-description ${nameError ? "create-app-name-error" : ""}`}
              autoFocus
            />
            <p id="create-app-name-description" className="text-xs text-gray-500 dark:text-gray-400">
              Leave this blank to use a generated name.
            </p>
            {nameError ? (
              <p id="create-app-name-error" className="text-xs text-red-600">
                {nameError}
              </p>
            ) : null}
          </div>

          <DialogFooter className="mt-6 flex-row justify-start gap-3 sm:justify-start">
            <LoadingButton type="submit" loading={isSaving} loadingText="Creating...">
              Create app
            </LoadingButton>
            <Button type="button" variant="outline" onClick={handleClose} disabled={isSaving}>
              Cancel
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
