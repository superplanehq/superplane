import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";

import { isCanvasNameAlreadyExistsError } from "./uniqueCanvasName";

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
    if (open) {
      setName("");
      setNameError("");
    }
  }, [open]);

  const handleClose = () => {
    if (isSaving) {
      return;
    }

    onClose();
  };

  const handleCreate = async () => {
    const trimmedName = name.trim();

    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }

    if (trimmedName.length > MAX_APP_NAME_LENGTH) {
      setNameError(`Name must be ${MAX_APP_NAME_LENGTH} characters or less`);
      return;
    }

    try {
      await onCreate(trimmedName);
    } catch (error) {
      if (isCanvasNameAlreadyExistsError(error)) {
        setNameError("An app with this name already exists");
      }
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleClose();
        }
      }}
    >
      <DialogContent showCloseButton={false} className="sm:max-w-md">
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void handleCreate();
          }}
        >
          <DialogHeader>
            <DialogTitle>Create app</DialogTitle>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="create-app-name-input">App name</Label>
              <Input
                id="create-app-name-input"
                data-testid="create-app-name-input"
                value={name}
                onChange={(event) => {
                  if (event.target.value.length <= MAX_APP_NAME_LENGTH) {
                    setName(event.target.value);
                  }

                  if (nameError) {
                    setNameError("");
                  }
                }}
                maxLength={MAX_APP_NAME_LENGTH}
                placeholder="Sentry Analysis"
                autoFocus
              />
              {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
            </div>
          </div>

          <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
            <LoadingButton
              type="submit"
              disabled={!name.trim()}
              loading={isSaving}
              loadingText="Creating..."
              data-testid="create-app-submit-button"
            >
              Create
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
