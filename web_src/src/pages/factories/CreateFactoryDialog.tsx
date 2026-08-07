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

interface CreateFactoryDialogProps {
  open: boolean;
  isSaving: boolean;
  onClose: () => void;
  onCreate: (input: { name: string; description: string }) => Promise<void>;
}

export function CreateFactoryDialog({ open, isSaving, onClose, onCreate }: CreateFactoryDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (open) {
      setName("");
      setDescription("");
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

    try {
      await onCreate({
        name: trimmedName,
        description: description.trim(),
      });
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to create workspace");
      showErrorToast(message);
      if (message.toLowerCase().includes("already") || message.toLowerCase().includes("exists")) {
        setNameError("A workspace with this name already exists");
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
          <DialogTitle>Create workspace</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="factory-name-input">Name</Label>
            <Input
              id="factory-name-input"
              data-testid="factory-name-input"
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
            <Label htmlFor="factory-description-input">Description</Label>
            <Textarea
              id="factory-description-input"
              data-testid="factory-description-input"
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
            onClick={() => void handleCreate()}
            disabled={!name.trim()}
            loading={isSaving}
            loadingText="Creating..."
            data-testid="factory-create-button"
          >
            Create
          </LoadingButton>
          <Button type="button" variant="outline" onClick={handleClose} disabled={isSaving}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
