import { ChevronDown } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import type { ColumnAutomationDisclosureState } from "@/hooks/useColumnAutomationCollapsePreference";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent } from "@/ui/collapsible";

export function ColumnAutomationToggle({
  title,
  disclosure,
}: {
  title: string;
  disclosure: ColumnAutomationDisclosureState;
}) {
  const label = `${disclosure.open ? "Collapse" : "Expand"} automations for ${title}`;
  return (
    <Button
      variant="ghost"
      size="xs"
      className="size-6 shrink-0 p-0 text-muted-foreground"
      aria-label={label}
      title={label}
      aria-expanded={disclosure.open}
      aria-controls={disclosure.contentId}
      onClick={() => disclosure.setOpen(!disclosure.open)}
    >
      <ChevronDown
        className={cn(
          "size-3.5 transition-transform duration-180 motion-reduce:transition-none",
          !disclosure.open && "-rotate-90",
        )}
        aria-hidden
      />
    </Button>
  );
}

export function ColumnAutomationDisclosure({
  disclosure,
  children,
}: {
  disclosure: ColumnAutomationDisclosureState;
  children: ReactNode;
}) {
  return (
    <Collapsible
      open={disclosure.open}
      onOpenChange={disclosure.setOpen}
      className="grid transition-[grid-template-rows,visibility] duration-180 motion-reduce:transition-none"
      style={{ gridTemplateRows: disclosure.open ? "1fr" : "0fr", visibility: disclosure.open ? "visible" : "hidden" }}
    >
      <CollapsibleContent
        forceMount
        id={disclosure.contentId}
        aria-hidden={!disclosure.open}
        inert={!disclosure.open}
        className="min-h-0 overflow-hidden"
      >
        <div className="px-3 pb-2.5">{children}</div>
      </CollapsibleContent>
    </Collapsible>
  );
}
