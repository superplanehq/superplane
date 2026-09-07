import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { DialogFooter } from "@/components/ui/dialog";
import { ExternalLink } from "lucide-react";

import type { OrganizationsBrowserAction } from "@/api-client";

interface IntegrationCreateDialogFooterProps {
  pendingWebhookSetup: { id: string; webhookUrl: string; config: Record<string, unknown> } | null;
  createIntegrationBrowserAction: OrganizationsBrowserAction | undefined;
  browserActionCompleted: boolean;
  mutationPending: boolean;
  isCreatePending: boolean;
  integrationName: string;
  /** Whether every visible required configuration field currently has a value. */
  requiredFieldsFilled: boolean;
  onCompleteWebhookSetup: () => Promise<void>;
  onBrowserActionContinue: () => void;
  onBrowserActionConfigSave: () => Promise<void>;
  onFinishBrowserActionSetup: () => Promise<void>;
  onSubmit: () => Promise<void>;
  onClose: () => void;
}

export function IntegrationCreateDialogFooter({
  pendingWebhookSetup,
  createIntegrationBrowserAction,
  browserActionCompleted,
  mutationPending,
  isCreatePending,
  integrationName,
  requiredFieldsFilled,
  onCompleteWebhookSetup,
  onBrowserActionContinue,
  onBrowserActionConfigSave,
  onFinishBrowserActionSetup,
  onSubmit,
  onClose,
}: IntegrationCreateDialogFooterProps) {
  if (pendingWebhookSetup) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        <LoadingButton
          color="blue"
          onClick={() => void onCompleteWebhookSetup()}
          disabled={!requiredFieldsFilled}
          loading={mutationPending}
          loadingText="Completing..."
          className="flex items-center gap-2"
        >
          Complete setup
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Done
        </Button>
      </DialogFooter>
    );
  }

  if (createIntegrationBrowserAction && !browserActionCompleted) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        {createIntegrationBrowserAction.url ? (
          <Button type="button" onClick={onBrowserActionContinue} className="flex items-center gap-2">
            <ExternalLink className="h-4 w-4" />
            Continue setup
          </Button>
        ) : (
          <LoadingButton
            color="blue"
            onClick={() => void onBrowserActionConfigSave()}
            loading={mutationPending}
            loadingText="Saving..."
          >
            Save
          </LoadingButton>
        )}
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Cancel
        </Button>
      </DialogFooter>
    );
  }

  if (createIntegrationBrowserAction) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        <LoadingButton
          color="blue"
          onClick={() => void onFinishBrowserActionSetup()}
          loading={mutationPending}
          loadingText="Saving..."
        >
          Done
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Cancel
        </Button>
      </DialogFooter>
    );
  }

  return (
    <DialogFooter className="gap-2 sm:justify-start mt-6">
      <LoadingButton
        color="blue"
        onClick={() => void onSubmit()}
        disabled={!integrationName?.trim() || !requiredFieldsFilled}
        loading={isCreatePending}
        loadingText="Connecting..."
        className="flex items-center gap-2"
      >
        Connect
      </LoadingButton>
      <Button variant="outline" onClick={onClose} disabled={isCreatePending}>
        Cancel
      </Button>
    </DialogFooter>
  );
}
