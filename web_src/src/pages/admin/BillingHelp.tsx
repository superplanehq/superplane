import { Button } from "@/components/ui/button";
import { CircleHelp } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export function BillingHelp({ label, text }: { label: string; text: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon-xs"
          type="button"
          aria-label={`Help for ${label}`}
          className="size-5 inline-flex align-middle rounded-full text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <CircleHelp className="size-3.5" aria-hidden="true" />
        </Button>
      </TooltipTrigger>
      <TooltipContent className="max-w-72 px-3 py-2 text-xs leading-relaxed" sideOffset={6}>
        {text}
      </TooltipContent>
    </Tooltip>
  );
}
