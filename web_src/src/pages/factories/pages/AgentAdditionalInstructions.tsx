import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useId } from "react";
import type { PlanningReviewDraft } from "./planningReviewMockup";

export function AgentAdditionalInstructions({
  draft,
  disabled,
  onChange,
  description,
  placeholder,
}: {
  description: string;
  placeholder: string;
  draft: PlanningReviewDraft;
  disabled: boolean;
  onChange: (draft: PlanningReviewDraft) => void;
}) {
  const id = useId();
  const component = draft.components[0];
  if (!component) return null;
  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor={id} className="text-base sm:text-sm">
        Additional instructions
      </Label>
      <p id={`${id}-help`} className="text-xs text-muted-foreground">
        {description}
      </p>
      <Textarea
        id={id}
        aria-describedby={`${id}-help`}
        disabled={disabled}
        rows={5}
        className="min-h-32 resize-y border-border bg-card shadow-none dark:border-border dark:bg-card"
        placeholder={placeholder}
        value={
          typeof component.configuration.additionalInstructions === "string"
            ? component.configuration.additionalInstructions
            : ""
        }
        onChange={(event) =>
          onChange({
            ...draft,
            components: draft.components.map((item) =>
              item.id === component.id
                ? { ...item, configuration: { ...item.configuration, additionalInstructions: event.target.value } }
                : item,
            ),
          })
        }
      />
    </div>
  );
}
