import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";
import {
  WORKSPACE_KEY_MAX_LENGTH,
  WORKSPACE_KEY_MIN_LENGTH,
  isValidWorkspaceKey,
  normalizeWorkspaceKey,
  suggestWorkspaceKeyFromName,
} from "./lib/workspaceKey";

const MAX_NAME_LENGTH = 128;
const MAX_DESCRIPTION_LENGTH = 500;

interface CreateFactoryDialogProps {
  open: boolean;
  isSaving: boolean;
  onClose: () => void;
  onCreate: (input: { name: string; description: string; key: string }) => Promise<void>;
}

export function CreateFactoryDialog({ open, isSaving, onClose, onCreate }: CreateFactoryDialogProps) {
  const {
    name,
    description,
    key,
    nameError,
    keyError,
    setDescription,
    handleNameChange,
    handleKeyChange,
    handleCreate,
    canSubmit,
  } = useCreateFactoryForm(open, onCreate);

  const handleClose = () => {
    if (isSaving) {
      return;
    }
    onClose();
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
                  handleNameChange(event.target.value);
                }
              }}
              maxLength={MAX_NAME_LENGTH}
              autoFocus
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="factory-key-input">Workspace key</Label>
            <Input
              id="factory-key-input"
              data-testid="factory-key-input"
              value={key}
              onChange={(event) => handleKeyChange(event.target.value)}
              maxLength={WORKSPACE_KEY_MAX_LENGTH}
              placeholder="SP"
              autoComplete="off"
              className="uppercase tracking-wider"
            />
            <p className="text-[11px] text-muted-foreground">
              Used to build work order IDs such as <span className="font-mono">{key || "SP"}-1</span>.{" "}
              {WORKSPACE_KEY_MIN_LENGTH}-{WORKSPACE_KEY_MAX_LENGTH} uppercase letters; must be unique in your
              organization.
            </p>
            {keyError ? <p className="text-xs text-red-600">{keyError}</p> : null}
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
            disabled={!canSubmit}
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

/** Form state for the create dialog, reset every time the dialog opens. */
function useCreateFactoryForm(open: boolean, onCreate: CreateFactoryDialogProps["onCreate"]) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [key, setKey] = useState("");
  const [keyEditedManually, setKeyEditedManually] = useState(false);
  const [nameError, setNameError] = useState("");
  const [keyError, setKeyError] = useState("");

  useEffect(() => {
    if (open) {
      setName("");
      setDescription("");
      setKey("");
      setKeyEditedManually(false);
      setNameError("");
      setKeyError("");
    }
  }, [open]);

  const handleNameChange = (nextName: string) => {
    setName(nextName);
    setNameError("");
    // Auto-suggest the key from the name until the user takes over.
    // Manual edits stick even if the user later clears them, matching
    // the expectation set by tools such as Linear and Jira.
    if (!keyEditedManually) {
      setKey(suggestWorkspaceKeyFromName(nextName));
      setKeyError("");
    }
  };

  const handleKeyChange = (nextKey: string) => {
    setKeyEditedManually(true);
    setKey(normalizeWorkspaceKey(nextKey));
    setKeyError("");
  };

  const handleCreate = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }
    if (!isValidWorkspaceKey(key)) {
      setKeyError(`Use ${WORKSPACE_KEY_MIN_LENGTH} to ${WORKSPACE_KEY_MAX_LENGTH} uppercase letters.`);
      return;
    }

    try {
      await onCreate({ name: trimmedName, description: description.trim(), key });
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to create workspace");
      showErrorToast(message);
      const lowered = message.toLowerCase();
      if (lowered.includes("key")) {
        setKeyError(message);
        return;
      }
      if (lowered.includes("already") || lowered.includes("exists")) {
        setNameError("A workspace with this name already exists");
      }
    }
  };

  return {
    name,
    description,
    key,
    nameError,
    keyError,
    setDescription,
    handleNameChange,
    handleKeyChange,
    handleCreate,
    canSubmit: Boolean(name.trim()) && isValidWorkspaceKey(key),
  };
}
