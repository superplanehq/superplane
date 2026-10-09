import { ArrowRight } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { useIsMobile } from "@/hooks/use-mobile";
import { cn } from "@/lib/utils";
import { Accordion, AccordionContent, AccordionItem } from "@/ui/accordion";
import type { DraftReadinessNote } from "../../lib/draftReadiness";
import { SPLIT_RUN_CHAT_COLUMN_CLASSNAME } from "./splitRunPopupModel";

export function PlanningImplementationControls({
  startDiscouraged = false,
  readinessNote,
  modelSelect,
  actions,
  canSend,
  showSuggestChanges,
  onSuggestChanges,
}: {
  startDiscouraged?: boolean;
  readinessNote?: DraftReadinessNote;
  modelSelect?: ReactNode;
  actions?: ReactNode;
  canSend: boolean;
  showSuggestChanges: boolean;
  onSuggestChanges: () => void;
}) {
  const isMobile = useIsMobile();
  if (!modelSelect && !actions) return null;
  const options = (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-2" data-testid="split-run-intent-settings">
      {modelSelect}
      {actions}
    </div>
  );
  return (
    <section aria-label="Implementation" className="shrink-0 border-t border-border py-3">
      <div className={cn(SPLIT_RUN_CHAT_COLUMN_CLASSNAME, "flex flex-wrap items-center justify-between gap-2")}>
        {startDiscouraged ? (
          isMobile ? (
            <PhoneOverrideDrawer note={readinessNote} modelSelect={modelSelect} actions={actions} />
          ) : (
            <WideDiscouragedOptions>{options}</WideDiscouragedOptions>
          )
        ) : (
          <>
            {showSuggestChanges ? (
              <Button
                type="button"
                variant="link"
                size="sm"
                className="px-0 text-[12px] font-normal text-muted-foreground underline underline-offset-4 hover:text-foreground"
                disabled={!canSend}
                onClick={onSuggestChanges}
              >
                Suggest changes
              </Button>
            ) : (
              <span className="text-[12px] text-muted-foreground">Implementation</span>
            )}
            {options}
          </>
        )}
      </div>
    </section>
  );
}

function WideDiscouragedOptions({ children }: { children: ReactNode }) {
  const [expanded, setExpanded] = useState(false);
  const optionsId = useId();
  return (
    <>
      <OverrideToggle expanded={expanded} optionsId={optionsId} onToggle={() => setExpanded((current) => !current)} />
      <div id={optionsId} aria-hidden={!expanded} inert={!expanded} className={cn("ml-auto", !expanded && "invisible")}>
        {children}
      </div>
    </>
  );
}

function PhoneOverrideDrawer({
  note,
  modelSelect,
  actions,
}: {
  note?: DraftReadinessNote;
  modelSelect?: ReactNode;
  actions?: ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);
  const optionsId = useId();
  return (
    <div className="flex w-full min-w-0 flex-col">
      <OverrideToggle expanded={expanded} optionsId={optionsId} onToggle={() => setExpanded((current) => !current)} />
      <Accordion type="single" collapsible value={expanded ? "override" : ""} onValueChange={() => undefined}>
        <AccordionItem value="override" className="border-0">
          <AccordionContent id={optionsId} inert={!expanded} aria-hidden={!expanded} className="pb-0">
            <div className="flex min-w-0 flex-col gap-3 pt-2" data-testid="phone-override-drawer">
              {note ? (
                <div className="min-w-0">
                  <h3 className="text-[13px] font-medium leading-5 break-words text-foreground">{note.headline}</h3>
                  {note.text ? (
                    <p className="mt-0.5 text-[12px] leading-4 break-words text-muted-foreground">{note.text}</p>
                  ) : null}
                </div>
              ) : null}
              <div className="flex w-full min-w-0 items-center justify-end gap-2" data-testid="split-run-intent-settings">
                {modelSelect}
                {actions}
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  );
}

function OverrideToggle({
  expanded,
  optionsId,
  onToggle,
}: {
  expanded: boolean;
  optionsId: string;
  onToggle: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px] text-muted-foreground">
      <span>Starting not recommended.</span>
      <Button
        type="button"
        variant="link"
        size="sm"
        className="h-auto gap-1 p-0 text-[12px] text-muted-foreground underline underline-offset-4 hover:text-foreground"
        aria-expanded={expanded}
        aria-controls={optionsId}
        onClick={onToggle}
      >
        {expanded ? "Hide options" : "Override"}
        {!expanded ? <ArrowRight aria-hidden className="size-3" /> : null}
      </Button>
    </div>
  );
}
