import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useState } from "react";

interface DispatchWorkOrderDialogProps {
  open: boolean;
  lines: FactoriesFactoryLine[];
  isSaving: boolean;
  canDispatch: boolean;
  onClose: () => void;
  onDispatch: (lineName: string) => Promise<void>;
}

export function DispatchWorkOrderDialog({
  open,
  lines,
  isSaving,
  canDispatch,
  onClose,
  onDispatch,
}: DispatchWorkOrderDialogProps) {
  const [lineName, setLineName] = useState("");

  useEffect(() => {
    if (open) {
      setLineName(lines[0]?.name ?? "");
    }
  }, [open, lines]);

  const handleClose = () => {
    if (isSaving) {
      return;
    }
    onClose();
  };

  const handleDispatch = async () => {
    if (!lineName) {
      showErrorToast("Select a line to dispatch to.");
      return;
    }

    try {
      await onDispatch(lineName);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to dispatch work order"));
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
          <DialogTitle>Dispatch to line</DialogTitle>
        </DialogHeader>

        {lines.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Configure at least one factory line before dispatching work orders.
          </p>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="dispatch-line-select">Line</Label>
            <Select value={lineName} onValueChange={setLineName}>
              <SelectTrigger id="dispatch-line-select" data-testid="dispatch-line-select">
                <SelectValue placeholder="Select a line" />
              </SelectTrigger>
              <SelectContent>
                {lines.map((line) => (
                  <SelectItem key={line.id ?? line.name} value={line.name ?? ""}>
                    {line.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
            <LoadingButton
              onClick={() => void handleDispatch()}
              disabled={!canDispatch || lines.length === 0 || !lineName}
              loading={isSaving}
              loadingText="Dispatching..."
              data-testid="dispatch-work-order-submit"
            >
              Dispatch
            </LoadingButton>
          </PermissionTooltip>
          <Button type="button" variant="outline" onClick={handleClose} disabled={isSaving}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
