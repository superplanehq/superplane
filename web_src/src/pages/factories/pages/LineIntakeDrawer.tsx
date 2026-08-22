import { Plus, XIcon } from "lucide-react";
import { useState } from "react";

import { AddIntakePicker } from "./AddIntakePicker";
import {
  intakeAutomationFixture,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
  type LineIntakeSource,
} from "./lineIntakeModel";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";

interface LineIntakeDrawerProps {
  onClose: () => void;
}

/**
 * Column beside the line board. Lists the automations that listen to
 * external sources, evaluate events, and create backlog work orders.
 */
export function LineIntakeDrawer({ onClose }: LineIntakeDrawerProps) {
  const [openSourceId, setOpenSourceId] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const openSource = openSourceId ? lineIntakeSourceById(openSourceId) : undefined;

  return (
    <>
      <aside
        className="flex h-full min-h-0 w-72 shrink-0 flex-col border-r border-border bg-slate-200 dark:bg-slate-800"
        data-testid="line-intake-drawer"
        aria-label="Intake"
      >
        <header className="flex shrink-0 items-center justify-between gap-2 px-3 pb-2 pt-3">
          <h2 className="workspace-section-title">Intake</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close Intake"
            title="Close Intake"
            data-testid="line-intake-close"
            className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <XIcon className="size-3.5" aria-hidden />
          </button>
        </header>
        <p className="workspace-body-text shrink-0 px-3 pb-3 text-muted-foreground">
          Automations that listen, evaluate, and create backlog work orders.
        </p>
        <ul className="flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto px-2 pb-3 [scrollbar-width:thin]">
          {LINE_INTAKE_SOURCES.map((source) => (
            <li key={source.id}>
              <IntakeSourceCard source={source} onOpen={() => setOpenSourceId(source.id)} />
            </li>
          ))}
          <li>
            <button
              type="button"
              onClick={() => setPickerOpen(true)}
              data-testid="line-intake-add"
              className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border bg-muted/60 px-3 py-2.5 text-[13px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:border-foreground/20 hover:bg-muted hover:text-foreground"
            >
              <Plus className="size-3.5 shrink-0" aria-hidden />
              Add intake
            </button>
          </li>
        </ul>
      </aside>

      <AddIntakePicker
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onSelect={(template) => {
          setPickerOpen(false);
          if (lineIntakeSourceById(template.id)) {
            setOpenSourceId(template.id);
          }
        }}
      />

      {openSource ? (
        <WorkOrderSplitRunPopup
          key={openSource.id}
          fixture={intakeAutomationFixture(openSource)}
          onClose={() => setOpenSourceId(null)}
          fixed
        />
      ) : null}
    </>
  );
}

function IntakeSourceCard({ source, onOpen }: { source: LineIntakeSource; onOpen: () => void }) {
  return (
    <article
      className="relative w-full rounded-lg bg-card px-3 py-2.5 shadow-sm"
      data-testid={`line-intake-source-${source.id}`}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 rounded-lg"
        aria-label={`Open ${source.name} intake`}
        onClick={onOpen}
      />
      <div className="relative z-10 pointer-events-none flex items-start gap-2.5">
        <img src={source.iconSrc} alt={source.iconAlt} className="mt-0.5 size-5 shrink-0" />
        <div className="min-w-0 flex-1">
          <h3 className="text-[13px] font-medium tracking-[-0.01em] leading-[19.5px] text-foreground">{source.name}</h3>
          <p className="workspace-body-text mt-0.5 text-muted-foreground">{source.description}</p>
        </div>
      </div>
    </article>
  );
}
