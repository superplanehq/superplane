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
import type { CanvasCardData } from "@/pages/home/types";

const SLUG_PATTERN = /^[a-z0-9][a-z0-9_-]*[a-z0-9]$|^[a-z0-9]$/;

interface ConvertToAppDialogProps {
  canvas: CanvasCardData;
  isOpen: boolean;
  onClose: () => void;
}

export function ConvertToAppDialog({ canvas, isOpen, onClose }: ConvertToAppDialogProps) {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { data: organization } = useOrganization(organizationId);
  const orgSlug = organization?.metadata?.name ?? organizationId;

  const deriveSlug = (name: string) =>
    name
      .toLowerCase()
      .replace(/[^a-z0-9_-]/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");

  const [displayName, setDisplayName] = useState(canvas.name ?? "");
  const [appSlug, setAppSlug] = useState(deriveSlug(canvas.name ?? ""));
  const [description, setDescription] = useState(canvas.description ?? "");
  const [slugError, setSlugError] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);

  const createMutation = useCreateApp(organizationId);

  const handleClose = () => {
    setSlugError("");
    setSubmitError(null);
    onClose();
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, "");
    setAppSlug(value);
    if (slugError) setSlugError("");
  };

  const validate = (): boolean => {
    if (!appSlug) {
      setSlugError("App slug is required.");
      return false;
    }
    if (!SLUG_PATTERN.test(appSlug)) {
      setSlugError("Slug must be lowercase letters, numbers, hyphens, or underscores.");
      return false;
    }
    return true;
  };

  const handleConvert = async () => {
    if (!validate()) return;
    setSubmitError(null);

    try {
      const app = await createMutation.mutateAsync({
        displayName: displayName.trim() || canvas.name,
        appSlug,
        description: description.trim() || undefined,
      });
      showSuccessToast(`Canvas converted to App "${displayName}"`);
      handleClose();
      if (app?.metadata?.id) {
        navigate(`/${organizationId}/apps/${app.metadata.id}`);
      }
    } catch (err) {
      setSubmitError(getApiErrorMessage(err) ?? "Failed to convert canvas. Please try again.");
      showErrorToast("Failed to convert canvas to App");
    }
  };

  const fullSlugPreview = appSlug ? `${orgSlug}-${appSlug}` : "";

  return (
    <Dialog size="md" open={isOpen} onClose={handleClose}>
      <DialogTitle>Convert to App</DialogTitle>
      <DialogDescription>
        Convert <strong>{canvas.name}</strong> to a SuperPlane App. The canvas YAML and dashboard will be exported to
        a new Code Storage repository.
      </DialogDescription>
      <DialogBody>
        <div className="space-y-4">
          <Field>
            <Label htmlFor="convert-display-name">Display name</Label>
            <Input
              id="convert-display-name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="My Payments Platform"
              autoFocus
            />
          </Field>

          <Field>
            <Label htmlFor="convert-app-slug">App slug</Label>
            <Input
              id="convert-app-slug"
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
            <Label htmlFor="convert-description">Description (optional)</Label>
            <Textarea
              id="convert-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of this app…"
              rows={3}
            />
          </Field>

          {submitError && <p className="text-sm text-red-500">{submitError}</p>}
        </div>
      </DialogBody>
      <DialogActions>
        <Button variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
          Cancel
        </Button>
        <Button onClick={handleConvert} disabled={createMutation.isPending}>
          {createMutation.isPending ? "Converting…" : "Convert to App"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
