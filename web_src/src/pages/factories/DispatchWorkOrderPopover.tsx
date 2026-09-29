import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useEffect, useState, type ReactNode } from "react";

interface DispatchWorkOrderPopoverProps {
  lines: FactoriesFactoryLine[];
  isSaving: boolean;
  canDispatch: boolean;
  align?: "start" | "center" | "end";
  submitLabel?: string;
  onDispatch: (input: { lineName: string }) => Promise<void>;
  children: ReactNode;
}

export function DispatchWorkOrderPopover({
  lines,
  isSaving,
  canDispatch,
  align = "end",
  submitLabel = "Dispatch",
  onDispatch,
  children,
}: DispatchWorkOrderPopoverProps) {
  const [open, setOpen] = useState(false);
  const [lineName, setLineName] = useState("");

  useEffect(() => {
    if (open) {
      setLineName(lines[0]?.name ?? "");
    }
  }, [open, lines]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }
    setOpen(nextOpen);
  };

  const handleDispatch = async () => {
    if (!lineName) {
      showErrorToast("Select a line to dispatch to.");
      return;
    }

    try {
      await onDispatch({ lineName });
      setOpen(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to dispatch task"));
    }
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange} modal={false}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="w-80 p-3" sideOffset={8}>
        <div className="space-y-3">
          {lines.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {submitLabel === "Start"
                ? "Configure at least one line before you start a task."
                : "Configure at least one line before dispatching tasks."}
            </p>
          ) : (
            <div className="space-y-1.5">
              <Label htmlFor="dispatch-line-select" className="text-xs">
                Line
              </Label>
              <Select value={lineName} onValueChange={setLineName}>
                <SelectTrigger id="dispatch-line-select" className="w-full" data-testid="dispatch-line-select">
                  <SelectValue placeholder="Select a line" />
                </SelectTrigger>
                <SelectContent position="popper">
                  {lines.map((line) => (
                    <SelectItem key={line.id ?? line.name} value={line.name ?? ""}>
                      {line.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch tasks.">
            <LoadingButton
              onClick={() => void handleDispatch()}
              disabled={!canDispatch || lines.length === 0 || !lineName}
              loading={isSaving}
              loadingText={submitLabel === "Start" ? "Starting..." : "Dispatching..."}
              className="w-full"
              data-testid="dispatch-work-order-submit"
            >
              {submitLabel}
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </PopoverContent>
    </Popover>
  );
}
