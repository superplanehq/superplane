import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Field, Label } from "@/components/Fieldset/fieldset";
import { Input } from "@/components/Input/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { useCreateApp } from "@/hooks/useAppData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { useOrganization } from "@/hooks/useOrganizationData";

const MAX_DISPLAY_NAME_LENGTH = 80;
const MAX_DESCRIPTION_LENGTH = 200;
const SLUG_PATTERN = /^[a-z0-9][a-z0-9_-]*[a-z0-9]$|^[a-z0-9]$/;

interface CreateAppDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export function CreateAppDialog({ isOpen, onClose }: CreateAppDialogProps) {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { data: organization } = useOrganization(organizationId);
  const orgSlug = organization?.metadata?.name ?? organizationId;

  const [displayName, setDisplayName] = useState("");
  const [appSlug, setAppSlug] = useState("");
  const [description, setDescription] = useState("");
  const [slugError, setSlugError] = useState("");
  const [nameError, setNameError] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);

  const createMutation = useCreateApp(organizationId);

  const resetForm = () => {
    setDisplayName("");
    setAppSlug("");
    setDescription("");
    setSlugError("");
    setNameError("");
    setSubmitError(null);
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const deriveSlugFromName = (name: string) => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9_-]/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");
  };

  const handleDisplayNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setDisplayName(value);
    if (!appSlug || appSlug === deriveSlugFromName(displayName)) {
      setAppSlug(deriveSlugFromName(value));
    }
    if (nameError) setNameError("");
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, "");
    setAppSlug(value);
    if (slugError) setSlugError("");
  };

  const validate = (): boolean => {
    let valid = true;
    if (!displayName.trim()) {
      setNameError("Display name is required.");
      valid = false;
    }
    if (!appSlug) {
      setSlugError("App slug is required.");
      valid = false;
    } else if (!SLUG_PATTERN.test(appSlug)) {
      setSlugError("Slug must be lowercase letters, numbers, hyphens, or underscores.");
      valid = false;
    }
    return valid;
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    setSubmitError(null);

    try {
      const app = await createMutation.mutateAsync({
        displayName: displayName.trim(),
        appSlug,
        description: description.trim() || undefined,
      });
      showSuccessToast(`App "${displayName}" created`);
      handleClose();
      if (app?.metadata?.id) {
        navigate(`/${organizationId}/apps/${app.metadata.id}`);
      }
    } catch (err) {
      setSubmitError(getApiErrorMessage(err) ?? "Failed to create app. Please try again.");
      showErrorToast("Failed to create app");
    }
  };

  const fullSlugPreview = appSlug ? `${orgSlug}-${appSlug}` : "";

  return (
    <Dialog size="md" open={isOpen} onClose={handleClose}>
      <DialogTitle>Create App</DialogTitle>
      <DialogDescription>Create a new App with a blank scaffold. A Code Storage repository will be provisioned automatically.</DialogDescription>
      <DialogBody>
        <div className="space-y-4">
          <Field>
            <Label htmlFor="app-display-name">Display name</Label>
            <Input
              id="app-display-name"
              value={displayName}
              onChange={handleDisplayNameChange}
              placeholder="My Payments Platform"
              maxLength={MAX_DISPLAY_NAME_LENGTH}
              autoFocus
            />
            {nameError && <p className="text-sm text-red-500 mt-1">{nameError}</p>}
          </Field>

          <Field>
            <Label htmlFor="app-slug">App slug</Label>
            <Input
              id="app-slug"
              value={appSlug}
              onChange={handleSlugChange}
              placeholder="my-payments-platform"
              className="font-mono"
            />
            {fullSlugPreview && (
              <p className="text-xs text-muted-foreground mt-1">
                Repo slug: <span className="font-mono">{fullSlugPreview}</span>
              </p>
            )}
            {slugError && <p className="text-sm text-red-500 mt-1">{slugError}</p>}
          </Field>

          <Field>
            <Label htmlFor="app-description">Description (optional)</Label>
            <Textarea
              id="app-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of this app…"
              maxLength={MAX_DESCRIPTION_LENGTH}
              rows={3}
            />
          </Field>

          {submitError && (
            <p className="text-sm text-red-500">{submitError}</p>
          )}
        </div>
      </DialogBody>
      <DialogActions>
        <Button variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} disabled={createMutation.isPending}>
          {createMutation.isPending ? "Creating…" : "Create App"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
