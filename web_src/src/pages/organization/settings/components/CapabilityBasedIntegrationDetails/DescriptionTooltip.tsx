import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Info } from "lucide-react";

export type DescriptionTooltipProps = {
  title: string;
  description: string | undefined;
};

export function DescriptionTooltip({ title, description }: DescriptionTooltipProps) {
  if (!description) {
    return null;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-content-muted hover:text-content-primary"
          aria-label={`About ${title}`}
        >
          <Info className="size-4 shrink-0" aria-hidden />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-xs text-balance">
        {description}
      </TooltipContent>
    </Tooltip>
  );
}
