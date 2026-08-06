import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Info } from "lucide-react";

export type FieldDescriptionTooltipProps = {
  title: string;
  description: string;
  /** Optional expression tip kept available when the inline quick tip is hidden. */
  expressionTip?: string;
};

export function FieldDescriptionTooltip({ title, description, expressionTip }: FieldDescriptionTooltipProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          aria-label={`About ${title}`}
        >
          <Info className="size-4 shrink-0" aria-hidden />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-xs space-y-1.5 text-balance">
        <p>{description}</p>
        {expressionTip ? <p>{expressionTip}</p> : null}
      </TooltipContent>
    </Tooltip>
  );
}
