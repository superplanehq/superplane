import { useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { automationsPath, linesPath } from "../lib/factoryPagePaths";
import { NewComponentButton, SelectComponentSidebar } from "./SelectComponentSidebar";
import { WorkOrderCanvas } from "./WorkOrderCanvas";

type ConfigureLocationState = {
  lineName?: string;
  phaseName?: string;
  automationName?: string;
};

/**
 * Full-bleed React Flow editor for a factory-line phase or a standalone automation.
 */
export function ConfigureAutomationPage() {
  const navigate = useNavigate();
  const { organizationId, factoryKey, lineId, phaseId, automationId } = useParams<{
    organizationId?: string;
    factoryKey?: string;
    lineId?: string;
    phaseId?: string;
    automationId?: string;
  }>();
  const location = useLocation();
  const state = (location.state as ConfigureLocationState | null) ?? {};
  const [pickerOpen, setPickerOpen] = useState(false);

  const isStandalone = Boolean(automationId);
  const backTo =
    organizationId && factoryKey
      ? isStandalone
        ? automationsPath(organizationId, factoryKey)
        : linesPath(organizationId, factoryKey)
      : isStandalone
        ? "/automations"
        : "/lines";
  const backLabel = isStandalone
    ? (state.automationName ?? automationId ?? "Automations")
    : (state.lineName ?? lineId ?? "Factory line");
  const title = isStandalone
    ? (state.automationName ?? automationId ?? "Automation")
    : (state.phaseName ?? phaseId ?? "Phase");

  function handleSave() {
    navigate(backTo);
  }

  function handleCancel() {
    navigate(backTo);
  }

  return (
    <div className="absolute inset-0 flex flex-col bg-background">
      <div className="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-3">
        <div className="min-w-0">
          <Link
            to={backTo}
            className="mb-2 inline-flex cursor-pointer items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-3.5" strokeWidth={1.75} />
            {backLabel}
          </Link>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">{title}</h1>
            <span className="rounded-md bg-foreground px-1.5 py-0.5 text-[11px] font-medium text-primary-foreground">
              Edit mode
            </span>
          </div>
          <p className="mt-1 text-[12px] text-muted-foreground">
            Drag steps and reconnect edges to configure this automation.
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2 pt-0.5">
          <Button type="button" variant="outline" size="sm" onClick={handleCancel}>
            Discard changes
          </Button>
          <Button type="button" size="sm" onClick={handleSave}>
            Save
          </Button>
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        <div className="relative min-h-0 min-w-0 flex-1">
          {!pickerOpen ? <NewComponentButton onClick={() => setPickerOpen(true)} /> : null}
          <WorkOrderCanvas editable />
        </div>
        {pickerOpen ? <SelectComponentSidebar onClose={() => setPickerOpen(false)} /> : null}
      </div>
    </div>
  );
}
