import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useEffect, useState, type ReactNode } from "react";

interface DispatchWorkOrderPopoverProps {
  lines: FactoriesFactoryLine[];
  isSaving: boolean;
  canDispatch: boolean;
  align?: "start" | "center" | "end";
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
  children: ReactNode;
}

export function DispatchWorkOrderPopover({
  lines,
  isSaving,
  canDispatch,
  align = "end",
  onDispatch,
  children,
}: DispatchWorkOrderPopoverProps) {
  const [open, setOpen] = useState(false);
  const [lineName, setLineName] = useState("");
  const [note, setNote] = useState("");

  useEffect(() => {
    if (open) {
      setLineName(lines[0]?.name ?? "");
      setNote("");
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
      const trimmedNote = note.trim();
      await onDispatch({ lineName, note: trimmedNote || undefined });
      setOpen(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to dispatch work order"));
    }
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange} modal={false}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="w-80 p-3" sideOffset={8}>
        <div className="space-y-3">
          {lines.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Configure at least one line before dispatching work orders.
            </p>
          ) : (
            <>
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

              <div className="space-y-1.5">
                <Label htmlFor="dispatch-note" className="text-xs">
                  Note (optional)
                </Label>
                <Textarea
                  id="dispatch-note"
                  value={note}
                  onChange={(event) => setNote(event.target.value)}
                  placeholder="Attempt 2 - dedicated canvas memory browser"
                  rows={2}
                  className="text-sm"
                  data-testid="dispatch-note-input"
                />
              </div>
            </>
          )}

          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
            <LoadingButton
              onClick={() => void handleDispatch()}
              disabled={!canDispatch || lines.length === 0 || !lineName}
              loading={isSaving}
              loadingText="Dispatching..."
              className="w-full"
              data-testid="dispatch-work-order-submit"
            >
              Dispatch
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </PopoverContent>
    </Popover>
  );
}
