import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";

const MAX_NAME_LENGTH = 128;
const MAX_DESCRIPTION_LENGTH = 500;

interface EditFactoryDialogProps {
  open: boolean;
  initialName: string;
  initialDescription?: string;
  isSaving: boolean;
  onClose: () => void;
  onSave: (input: { name: string; description: string }) => Promise<void>;
}

export function EditFactoryDialog({
  open,
  initialName,
  initialDescription = "",
  isSaving,
  onClose,
  onSave,
}: EditFactoryDialogProps) {
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

    try {
      await onSave({
        name: trimmedName,
        description: description.trim(),
      });
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to update factory");
      showErrorToast(message);
      if (message.toLowerCase().includes("already") || message.toLowerCase().includes("exists")) {
        setNameError("A factory with this name already exists");
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
          <DialogTitle>Edit factory</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-factory-name-input">Name</Label>
            <Input
              id="edit-factory-name-input"
              data-testid="edit-factory-name-input"
              value={name}
              onChange={(event) => {
                if (event.target.value.length <= MAX_NAME_LENGTH) {
                  setName(event.target.value);
                }
                if (nameError) {
                  setNameError("");
                }
              }}
              maxLength={MAX_NAME_LENGTH}
              autoFocus
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-factory-description-input">Description</Label>
            <Textarea
              id="edit-factory-description-input"
              data-testid="edit-factory-description-input"
              value={description}
              onChange={(event) => {
                if (event.target.value.length <= MAX_DESCRIPTION_LENGTH) {
                  setDescription(event.target.value);
                }
              }}
              maxLength={MAX_DESCRIPTION_LENGTH}
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
            data-testid="factory-edit-save-button"
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
