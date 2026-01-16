import { useEffect, useState } from "react";
import { showErrorToast } from "../../utils/toast";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "../Dialog/dialog";
import { Field, Label } from "../Fieldset/fieldset";
import { Icon } from "../Icon";
import { Input } from "../Input/input";
import { Textarea } from "../ui/textarea";
import { Button } from "../ui/button";

interface CreateCanvasModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { name: string; description?: string }) => Promise<void>;
  isLoading?: boolean;
  initialData?: { name: string; description?: string };
  mode?: "create" | "edit";
}

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false,
  initialData,
  mode = "create",
}: CreateCanvasModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (isOpen) {
      setName(initialData?.name ?? "");
      setDescription(initialData?.description ?? "");
      setNameError("");
    }
  }, [isOpen, initialData?.name, initialData?.description]);

  const handleClose = () => {
    setName("");
    setDescription("");
    setNameError("");
    onClose();
  };

  const handleSubmit = async () => {
    setNameError("");

    if (!name.trim()) {
      setNameError("Name is required");
      return;
    }

    if (name.trim().length > MAX_CANVAS_NAME_LENGTH) {
      setNameError(`Name must be ${MAX_CANVAS_NAME_LENGTH} characters or less`);
      return;
    }

    try {
      await onSubmit({
        name: name.trim(),
        description: description.trim() || undefined,
      });

      // Reset form and close modal
      setName("");
      setDescription("");
      setNameError("");
      onClose();
    } catch (error) {
      console.error("Error creating canvas:", error);
      const errorMessage = (error as Error)?.message || error?.toString() || "Failed to create canvas";

      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>{mode === "edit" ? "Edit canvas" : "New canvas"}</DialogTitle>
      <DialogDescription className="text-sm !text-[var(--color-gray-800)]">
        {mode === "edit"
          ? "Update the canvas details to keep things clear for your teammates."
          : "Create a new canvas to orchestrate your DevOps work. You can tweak the details any time."}
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <Icon name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          <Field>
            <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Canvas name *</Label>
            <Input
              data-testid="canvas-name-input"
              type="text"
              autoComplete="off"
              value={name}
              onChange={(e) => {
                if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                  setName(e.target.value);
                }
                if (nameError) {
                  setNameError("");
                }
              }}
              placeholder=""
              className={`w-full ${nameError ? "border-red-500" : ""}`}
              autoFocus
              maxLength={MAX_CANVAS_NAME_LENGTH}
            />
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {name.length}/{MAX_CANVAS_NAME_LENGTH} characters
            </div>
            {nameError && <div className="text-xs text-red-600 mt-1">{nameError}</div>}
          </Field>

          <Field>
            <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Description</Label>
            <Textarea
              value={description}
              onChange={(e) => {
                if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                  setDescription(e.target.value);
                }
              }}
              placeholder="Describe what is does (optional)"
              rows={3}
              className="w-full"
              maxLength={MAX_CANVAS_DESCRIPTION_LENGTH}
            />
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {description.length}/{MAX_CANVAS_DESCRIPTION_LENGTH} characters
            </div>
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button
          onClick={handleSubmit}
          disabled={!name.trim() || isLoading || !!nameError}
          className="flex items-center gap-2"
        >
          {mode === "edit"
            ? isLoading
              ? "Saving..."
              : "Save changes"
            : isLoading
              ? "Creating canvas..."
              : "Create canvas"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
