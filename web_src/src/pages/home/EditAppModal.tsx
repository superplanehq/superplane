import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

const MAX_APP_NAME_LENGTH = 50;
const MAX_APP_DESCRIPTION_LENGTH = 200;

interface EditAppModalProps {
  open: boolean;
  initialName: string;
  initialDescription?: string;
  isSaving: boolean;
  onClose: () => void;
  onSave: (data: { name: string; description: string }) => Promise<void>;
}

export function EditAppModal({
  open,
  initialName,
  initialDescription = "",
  isSaving,
  onClose,
  onSave,
}: EditAppModalProps) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (open) {
      setName(initialName);
      setDescription(initialDescription);
      setNameError("");
    }
  }, [open, initialDescription, initialName]);

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
      await onSave({
        name: trimmedName,
        description: description.trim(),
      });
    } catch (error) {
      const errorMessage = getApiErrorMessage(error, "Failed to update app");
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
          <DialogTitle>Edit app</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-app-name-input">App name</Label>
            <Input
              id="edit-app-name-input"
              data-testid="edit-app-name-input"
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
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-app-description-input">Description</Label>
            <Textarea
              id="edit-app-description-input"
              data-testid="edit-app-description-input"
              value={description}
              onChange={(event) => {
                if (event.target.value.length <= MAX_APP_DESCRIPTION_LENGTH) {
                  setDescription(event.target.value);
                }
              }}
              maxLength={MAX_APP_DESCRIPTION_LENGTH}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            onClick={() => void handleSave()}
            disabled={!name.trim()}
            loading={isSaving}
            loadingText="Saving..."
            data-testid="edit-app-save-button"
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
