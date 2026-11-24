import { cloneElement, type ComponentProps, type ReactElement, type ReactNode } from "react";
import { PlanningImplementationControls } from "./PlanningImplementationControls";
import type { SplitRunReview } from "./SplitRunReview";

/** Reuse task dispatch and its guards without the built-in planning confirmation. */
export function AutomationChatFooter({
  review,
  modelSelect,
}: {
  review: ReactElement<ComponentProps<typeof SplitRunReview>>;
  modelSelect?: ReactNode;
}) {
  const footer = review.props.footer;
  if (footer.kind !== "draft") return null;
  return (
    <PlanningImplementationControls
      modelSelect={modelSelect}
      actions={cloneElement(review, {
        footer: {
          ...footer,
          actions: footer.actions.map((action) =>
            action.kind === "start" ? { ...action, label: "Continue" } : action,
          ),
        },
        actionsOnly: true,
        modelSelect: undefined,
        confirmUnclearStart: false,
      })}
    />
  );
}
