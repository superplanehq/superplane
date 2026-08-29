import type { FactoriesFactoryLine, FactoriesWorkOrderLineDispatch } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDown, Plus, Sparkles } from "lucide-react";
import { Link } from "react-router";

import { DispatchWorkOrderPopover } from "../DispatchWorkOrderPopover";
import { factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  deriveFactoryLineRows,
  FACTORY_LINE_TONE_DOT_CLASS,
  type FactoryLineRowModel,
} from "../lib/workOrderFactoryLineRows";
import { SidebarSectionHeading } from "./SidebarPrimitives";

interface WorkOrderSidebarFactoryLinesProps {
  organizationId: string;
  factoryKey: string;
  lineDispatches: FactoriesWorkOrderLineDispatch[];
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  permissionsLoading: boolean;
  isDispatchable: boolean;
  isDispatching: boolean;
  onDispatch: (input: { lineName: string }) => Promise<void>;
}

export function WorkOrderSidebarFactoryLines({
  organizationId,
  factoryKey,
  lineDispatches,
  factoryLines,
  canDispatch,
  permissionsLoading,
  isDispatchable,
  isDispatching,
  onDispatch,
}: WorkOrderSidebarFactoryLinesProps) {
  const rows = deriveFactoryLineRows(lineDispatches);
  const isDispatchDisabled = !canDispatch || factoryLines.length === 0;

  return (
    <section>
      <SidebarSectionHeading>Factory Lines</SidebarSectionHeading>

      <div className="mt-2">
        {rows.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">Not run on a line yet.</p>
        ) : (
          rows.map((row) => (
            <FactoryLineRow key={row.lineId} row={row} organizationId={organizationId} factoryKey={factoryKey} />
          ))
        )}

        {isDispatchable ? (
          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch tasks.">
            <DispatchWorkOrderPopover
              lines={factoryLines}
              isSaving={isDispatching}
              canDispatch={canDispatch || permissionsLoading}
              onDispatch={onDispatch}
              align="start"
            >
              <Button
                type="button"
                variant="ghost"
                disabled={isDispatchDisabled}
                aria-label="Send to line"
                className={cn(
                  "-mx-1.5 mt-0.5 h-auto w-[calc(100%+0.75rem)] justify-start gap-2 whitespace-normal rounded-md px-1.5 py-1.5 text-left text-[13px] tracking-[-0.01em]",
                  "text-muted-foreground hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60",
                )}
                data-testid="work-order-sidebar-dispatch-button"
              >
                <Plus className="size-3.5 shrink-0" aria-hidden />
                Send to line
                <ChevronDown className="ml-auto size-3.5 shrink-0 text-muted-foreground" aria-hidden />
              </Button>
            </DispatchWorkOrderPopover>
          </PermissionTooltip>
        ) : null}
      </div>
    </section>
  );
}

function FactoryLineRow({
  row,
  organizationId,
  factoryKey,
}: {
  row: FactoryLineRowModel;
  organizationId: string;
  factoryKey: string;
}) {
  const href =
    row.lineId && row.lineId !== "unknown" ? factoryLineDetailPath(organizationId, factoryKey, row.lineId) : undefined;
  const inner = (
    <>
      <Sparkles className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <span className={cn("size-1.5 shrink-0 rounded-full", FACTORY_LINE_TONE_DOT_CLASS[row.tone])} aria-hidden />
      <span className="min-w-0 truncate text-[13px] tracking-[-0.01em] text-foreground">{row.lineName}</span>
    </>
  );
  const commonClass =
    "flex w-full items-center justify-start gap-2 whitespace-normal rounded-md py-1.5 text-left transition-colors hover:bg-accent/60 focus-visible:bg-accent/60";

  if (!href) {
    return <span className={cn(commonClass, "cursor-default")}>{inner}</span>;
  }

  return (
    <Link to={href} className={commonClass} aria-label={`View ${row.lineName}`}>
      {inner}
    </Link>
  );
}
