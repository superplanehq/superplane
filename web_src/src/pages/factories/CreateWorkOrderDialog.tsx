import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

interface CreateWorkOrderDialogProps {
  open: boolean;
  isSaving: boolean;
  onClose: () => void;
  onCreate: (input: { title: string; description: string }) => Promise<void>;
}

export function CreateWorkOrderDialog({ open, isSaving, onClose, onCreate }: CreateWorkOrderDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [titleError, setTitleError] = useState("");

  useEffect(() => {
    if (open) {
      setTitle("");
      setDescription("");
      setTitleError("");
    }
  }, [open]);

  const handleClose = () => {
    if (isSaving) {
      return;
    }
    onClose();
  };

  const handleCreate = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setTitleError("Title is required");
      return;
    }

    try {
      await onCreate({
        title: trimmedTitle,
        description: description.trim(),
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create work order"));
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
          <DialogTitle>Create work order</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="work-order-title-input">Title</Label>
            <Input
              id="work-order-title-input"
              data-testid="work-order-title-input"
              value={title}
              onChange={(event) => {
                if (event.target.value.length <= MAX_TITLE_LENGTH) {
                  setTitle(event.target.value);
                }
                if (titleError) {
                  setTitleError("");
                }
              }}
              maxLength={MAX_TITLE_LENGTH}
              autoFocus
            />
            {titleError ? <p className="text-xs text-red-600">{titleError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="work-order-description-input">Description</Label>
            <Textarea
              id="work-order-description-input"
              data-testid="work-order-description-input"
              value={description}
              onChange={(event) => {
                if (event.target.value.length <= MAX_DESCRIPTION_LENGTH) {
                  setDescription(event.target.value);
                }
              }}
              maxLength={MAX_DESCRIPTION_LENGTH}
              rows={4}
            />
          </div>
        </div>

        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            onClick={() => void handleCreate()}
            disabled={!title.trim()}
            loading={isSaving}
            loadingText="Creating..."
            data-testid="work-order-create-button"
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
