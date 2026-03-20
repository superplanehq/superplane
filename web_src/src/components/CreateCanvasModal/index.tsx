import { useEffect, useState } from "react";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { getApiErrorMessage } from "@/utils/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/utils/usageLimits";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
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
  onSubmit: (data: { name: string; description?: string; templateId?: string }) => Promise<void>;
  isLoading?: boolean;
  organizationId?: string;
  initialData?: { name: string; description?: string };
  templates?: { id: string; name: string; description?: string }[];
  defaultTemplateId?: string;
  mode?: "create" | "edit";
  fromTemplate?: boolean;
}

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false,
  organizationId,
  initialData,
  defaultTemplateId,
  mode = "create",
  fromTemplate = false,
}: CreateCanvasModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [submitError, setSubmitError] = useState<unknown>(null);

  useEffect(() => {
    if (isOpen) {
      setName(initialData?.name ?? "");
      setDescription(initialData?.description ?? "");
      setNameError("");
      setSubmitError(null);
    }
    if (isOpen && mode === "create") {
      setTemplateId(defaultTemplateId || "");
    }
    if (isOpen && mode !== "create") {
      setTemplateId("");
    }
  }, [isOpen, initialData?.name, initialData?.description, defaultTemplateId, mode]);

  const handleClose = () => {
    setName("");
    setDescription("");
    setNameError("");
    setTemplateId("");
    setSubmitError(null);
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
      setSubmitError(null);
      await onSubmit({
        name: name.trim(),
        description: description.trim() || undefined,
        templateId: templateId || undefined,
      });

      // Reset form and close modal
      setName("");
      setDescription("");
      setNameError("");
      setTemplateId("");
      onClose();
    } catch (error) {
      const errorMessage = getApiErrorMessage(error, "Failed to create canvas");
      setSubmitError(error);
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create canvas"));
      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  const usageLimitNotice = submitError ? getUsageLimitNotice(submitError, organizationId) : null;

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>
        {fromTemplate ? "New Canvas from template" : mode === "edit" ? "Edit Canvas" : "New Canvas"}
      </DialogTitle>
      <DialogDescription className="text-sm !text-[var(--color-gray-800)]">
        {fromTemplate
          ? "Create a canvas from this template. Give it a name and optional description to get started."
          : mode === "edit"
            ? "Update the canvas details to keep things clear for your teammates."
            : "Create a new canvas to orchestrate your DevOps work. You can tweak the details any time."}
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <Icon name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          {usageLimitNotice ? <UsageLimitAlert notice={usageLimitNotice} /> : null}
          {submitError && !usageLimitNotice ? (
            <Alert variant="destructive">
              <AlertTitle>Unable to save canvas</AlertTitle>
              <AlertDescription>{getApiErrorMessage(submitError, "Failed to create canvas")}</AlertDescription>
            </Alert>
          ) : null}
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
                if (submitError) {
                  setSubmitError(null);
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
            : fromTemplate
              ? isLoading
                ? "Creating Canvas"
                : "Create Canvas"
              : isLoading
                ? "Creating canvas..."
                : "Create canvas"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
