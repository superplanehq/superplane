import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

const MAX_APP_NAME_LENGTH = 50;

interface CreateAppModalProps {
  open: boolean;
  defaultName: string;
  isSaving: boolean;
  onClose: () => void;
  onSave: (name: string) => Promise<void>;
}

export function CreateAppModal({ open, defaultName, isSaving, onClose, onSave }: CreateAppModalProps) {
  const [name, setName] = useState(defaultName);
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (open) {
      setName(defaultName);
      setNameError("");
    }
  }, [open, defaultName]);

  const handleClose = () => {
    if (isSaving) {
      return;
    }

    onClose();
  };

  const handleSave = async () => {
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
      await onSave(trimmedName);
    } catch (error) {
      const errorMessage = getApiErrorMessage(error, "Failed to create app");
      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
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
        <DialogHeader>
          <DialogTitle>New App</DialogTitle>
        </DialogHeader>

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
            autoFocus
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                void handleSave();
              }
            }}
          />
          {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
        </div>

        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            onClick={() => void handleSave()}
            disabled={!name.trim()}
            loading={isSaving}
            loadingText="Saving..."
            data-testid="create-app-save-button"
          >
            Save
          </LoadingButton>
          <Button type="button" variant="outline" onClick={handleClose} disabled={isSaving}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
