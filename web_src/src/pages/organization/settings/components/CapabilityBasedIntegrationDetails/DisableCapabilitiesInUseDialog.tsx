import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Link } from "@/components/Link/link";
import { appPath } from "@/lib/appPaths";
import { LoadingButton } from "@/components/ui/loading-button";
import { Fragment } from "react";
import {
  INTEGRATION_INLINE_CODE_CLASSES,
  type CapabilityDisableCanvasRow,
  type CapabilityDisableCanvasSummary,
} from "./lib";

const MAX_CANVAS_NAMES_SHOWN = 3;

const CANVAS_LINK_CLASSES =
  "rounded-xs text-gray-900 hover:text-black dark:text-gray-100 dark:hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400/45 focus-visible:ring-offset-2 dark:focus-visible:ring-gray-500/40";

const CANVAS_LINK_UNDERLINE_STYLE = {
  textDecoration: "underline",
  textUnderlineOffset: "2px",
} as const;

function CanvasNamesUsedInSummary({
  organizationId,
  canvases,
}: {
  organizationId: string;
  canvases: CapabilityDisableCanvasSummary[];
}) {
  if (canvases.length === 0) {
    return <span className="text-sm text-gray-500 dark:text-gray-400">—</span>;
  }

  const shown = canvases.slice(0, MAX_CANVAS_NAMES_SHOWN);
  const restCount = canvases.length - shown.length;

  return (
    <span className="inline leading-relaxed text-sm text-gray-800 dark:text-gray-200">
      <span className="text-gray-600 dark:text-gray-400">used in </span>
      {shown.map((canvas, index) => (
        <Fragment key={canvas.canvasId}>
          {index > 0 ? <span className="text-gray-500 dark:text-gray-400">, </span> : null}
          <Link
            href={appPath(organizationId, canvas.canvasId)}
            target="_blank"
            rel="noopener noreferrer"
            style={CANVAS_LINK_UNDERLINE_STYLE}
            className={CANVAS_LINK_CLASSES}
          >
            {canvas.canvasName}
          </Link>
        </Fragment>
      ))}
      {restCount > 0 ? <span className="text-gray-600 dark:text-gray-400"> + {restCount}</span> : null}
    </span>
  );
}

export interface DisableCapabilitiesInUseDialogProps {
  organizationId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  capabilityDisableCanvasRows: CapabilityDisableCanvasRow[];
  canUpdateIntegrations: boolean;
  capabilitiesMutationPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DisableCapabilitiesInUseDialog({
  organizationId,
  open,
  onOpenChange,
  capabilityDisableCanvasRows,
  canUpdateIntegrations,
  capabilitiesMutationPending,
  onConfirm,
  onCancel,
}: DisableCapabilitiesInUseDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="w-[calc(100vw-2rem)] max-w-3xl">
        <DialogHeader>
          <DialogTitle>Capabilities still used on a canvas</DialogTitle>
          <DialogDescription asChild>
            <div className="space-y-3">
              <p className="text-sm text-gray-500 dark:text-gray-400">
                You are disabling capabilities that are in use on at least one canvas:
              </p>
              <div className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-700">
                <div className="overflow-x-auto">
                  <table className="w-full table-auto divide-y divide-gray-200 dark:divide-gray-800">
                    <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                      {capabilityDisableCanvasRows.map((row) => (
                        <tr key={row.capabilityName}>
                          <td className="whitespace-nowrap px-4 py-3 align-middle">
                            <code className={INTEGRATION_INLINE_CODE_CLASSES}>{row.capabilityName}</code>
                          </td>
                          <td className="min-w-0 px-4 py-3 align-middle">
                            <div className="min-w-0 break-words">
                              <CanvasNamesUsedInSummary organizationId={organizationId} canvases={row.canvases} />
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Those canvases may stop working as intended until you remove or replace the affected nodes.
              </p>
            </div>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            type="button"
            color="blue"
            onClick={onConfirm}
            disabled={!canUpdateIntegrations}
            loading={capabilitiesMutationPending}
            loadingText="Updating…"
          >
            Update capabilities
          </LoadingButton>
          <Button type="button" variant="outline" onClick={onCancel} disabled={capabilitiesMutationPending}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
