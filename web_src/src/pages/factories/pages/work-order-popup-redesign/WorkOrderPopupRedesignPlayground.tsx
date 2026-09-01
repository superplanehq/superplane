import { useState } from "react";

import { ConceptJobPopup } from "./ConceptJobPopup";
import { ConceptSessionPopup } from "./ConceptSessionPopup";
import { ConceptTracePopup } from "./ConceptTracePopup";
import { PrototypeSwitcher, RunOverlayBoardBackdrop } from "./popupShared";
import { AGENT_WORK_POPUP, type PopupConcept, type PopupFixture } from "./workOrderPopupMocks";

const CONCEPTS: { id: PopupConcept; label: string; pattern: string }[] = [
  { id: "session", label: "Session", pattern: "Agent feed" },
  { id: "trace", label: "Trace", pattern: "Span tree" },
  { id: "job", label: "Job", pattern: "Run report" },
];

/**
 * Storybook playground: line board under a popup that treats the task
 * as agent work. Production peek is unchanged.
 */
export function WorkOrderPopupRedesignPlayground({
  initialConcept = "session",
  fixture = AGENT_WORK_POPUP,
}: {
  initialConcept?: PopupConcept;
  fixture?: PopupFixture;
}) {
  const [concept, setConcept] = useState<PopupConcept>(initialConcept);
  const caption = CONCEPTS.find((entry) => entry.id === concept)?.pattern ?? "";

  return (
    <div className="relative min-h-svh">
      <RunOverlayBoardBackdrop />
      {concept === "session" ? <ConceptSessionPopup fixture={fixture} /> : null}
      {concept === "trace" ? <ConceptTracePopup fixture={fixture} /> : null}
      {concept === "job" ? <ConceptJobPopup fixture={fixture} /> : null}
      <PrototypeSwitcher value={concept} onChange={(id) => setConcept(id as PopupConcept)} options={CONCEPTS} />
      <p className="sr-only">{caption}</p>
    </div>
  );
}
