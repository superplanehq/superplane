import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

export interface IntegrationSetupFlowChoiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  integrationDefinition: IntegrationsIntegrationDefinition | null | undefined;
  onChooseGuided: () => void;
  onChooseClassic: () => void;
}

export function IntegrationSetupFlowChoiceDialog({
  open,
  onOpenChange,
  integrationDefinition,
  onChooseGuided,
  onChooseClassic,
}: IntegrationSetupFlowChoiceDialogProps) {
  const displayName =
    integrationDefinition?.label ||
    getIntegrationTypeDisplayName(undefined, integrationDefinition?.name) ||
    integrationDefinition?.name ||
    "Integration";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <IntegrationIcon
              integrationName={integrationDefinition?.name}
              iconSlug={integrationDefinition?.icon}
              className="h-8 w-8 shrink-0 text-gray-500 dark:text-gray-400"
            />
            <DialogTitle className="text-left">Choose how to connect</DialogTitle>
          </div>
          <DialogDescription>
            {displayName} supports guided setup (step-by-step) and the classic connect flow (one form). Pick the option
            you prefer.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex-col gap-2 sm:flex-col sm:justify-stretch sm:space-x-0">
          <Button type="button" className="w-full" onClick={() => onChooseGuided()}>
            Use guided setup
          </Button>
          <Button type="button" variant="outline" className="w-full" onClick={() => onChooseClassic()}>
            Use classic setup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
