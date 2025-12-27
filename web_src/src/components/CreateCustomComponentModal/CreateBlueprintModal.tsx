import { useState } from "react";
import { showErrorToast } from "../../utils/toast";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "../Dialog/dialog";
import { Field, Label } from "../Fieldset/fieldset";
import { Icon } from "../Icon";
import { Input } from "../Input/input";
import { Textarea } from "../Textarea/textarea";
import { Button } from "../ui/button";

interface CreateCustomComponentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { name: string; description?: string }) => Promise<void>;
  isLoading?: boolean;
}

const MAX_BLUEPRINT_NAME_LENGTH = 50;
const MAX_BLUEPRINT_DESCRIPTION_LENGTH = 200;

export function CreateCustomComponentModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false,
}: CreateCustomComponentModalProps) {
  const [blueprintName, setBlueprintName] = useState("");
  const [blueprintDescription, setBlueprintDescription] = useState("");
  const [nameError, setNameError] = useState("");

  const handleClose = () => {
    setBlueprintName("");
    setBlueprintDescription("");
    setNameError("");
    onClose();
  };

  const handleSubmit = async () => {
    setNameError("");

    if (!blueprintName.trim()) {
      setNameError("Bundle name is required");
      return;
    }

    if (blueprintName.trim().length > MAX_BLUEPRINT_NAME_LENGTH) {
      setNameError(`Bundle name must be ${MAX_BLUEPRINT_NAME_LENGTH} characters or less`);
      return;
    }

    try {
      await onSubmit({
        name: blueprintName.trim(),
        description: blueprintDescription.trim() || undefined,
      });

      // Reset form and close modal
      setBlueprintName("");
      setBlueprintDescription("");
      setNameError("");
      onClose();
    } catch (error) {
      console.error("Error creating Bundle:", error);
      const errorMessage =
        (error as Error)?.message || error?.toString() || "Something went wrong. We failed to create a Bundle";

      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A Bundle with this name already exists");
      }
    }
  };

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>New Bundle</DialogTitle>
      <DialogDescription className="text-sm">
        Create a custom Bundle that can be reused across your canvases and automations.
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <Icon name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          {/* Blueprint Name */}
          <Field>
            <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Bundle name *</Label>
            <Input
              data-testid="component-name-input"
              type="text"
              autoComplete="off"
              value={blueprintName}
              onChange={(e) => {
                if (e.target.value.length <= MAX_BLUEPRINT_NAME_LENGTH) {
                  setBlueprintName(e.target.value);
                }
                if (nameError) {
                  setNameError("");
                }
              }}
              placeholder="Give this Bundle a clear name for reuse"
              className={`w-full ${nameError ? "border-red-500" : ""}`}
              autoFocus
              maxLength={MAX_BLUEPRINT_NAME_LENGTH}
            />
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {blueprintName.length}/{MAX_BLUEPRINT_NAME_LENGTH} characters
            </div>
            {nameError && <div className="text-xs text-red-600 mt-1">{nameError}</div>}
          </Field>

          {/* Blueprint Description */}
          <Field>
            <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Description</Label>
            <Textarea
              value={blueprintDescription}
              onChange={(e) => {
                if (e.target.value.length <= MAX_BLUEPRINT_DESCRIPTION_LENGTH) {
                  setBlueprintDescription(e.target.value);
                }
              }}
              placeholder="Note the purpose of this Bundle (optional)"
              rows={3}
              className="w-full"
              maxLength={MAX_BLUEPRINT_DESCRIPTION_LENGTH}
            />
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {blueprintDescription.length}/{MAX_BLUEPRINT_DESCRIPTION_LENGTH} characters
            </div>
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button
          onClick={handleSubmit}
          disabled={!blueprintName.trim() || isLoading || !!nameError}
          className="flex items-center gap-2"
        >
          {isLoading ? "Creating Bundle..." : "Create Bundle"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
