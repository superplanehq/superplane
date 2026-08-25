import { RunCheckGrid } from "../work-order-run-overlay/runOverlayShared";
import {
  AgentLogList,
  DescriptionFileCard,
  OutputList,
  OwnerTimeCostRow,
  PopupBody,
  PopupHeader,
  PopupShell,
  SectionTitle,
  WaitingNotes,
} from "./popupShared";
import type { PopupFixture } from "./workOrderPopupMocks";

/**
 * Agent session. Pattern: Devin task feed.
 *
 * One column. The spec is a file. Outputs, scores, and the agent log follow.
 */
export function ConceptSessionPopup({ fixture }: { fixture: PopupFixture }) {
  return (
    <PopupShell testId="work-order-popup-session">
      <PopupHeader title={fixture.title}>
        <OwnerTimeCostRow fixture={fixture} />
      </PopupHeader>
      <PopupBody>
        <DescriptionFileCard artifact={fixture.description} />

        <section className="mt-8">
          <SectionTitle>Outputs</SectionTitle>
          <div className="mt-2">
            <OutputList artifacts={fixture.outputs} />
          </div>
        </section>

        <section className="mt-8">
          <WaitingNotes notes={fixture.waitingNotes} />
        </section>

        <section className="mt-8">
          <SectionTitle>Scores</SectionTitle>
          <RunCheckGrid checks={fixture.checks} emptyLabel="No scores yet." className="mt-3" />
        </section>

        <section className="mt-8">
          <SectionTitle>Worked for {fixture.elapsed}</SectionTitle>
          <p className="workspace-body-text mt-1 text-muted-foreground">Agent and automation steps. No comments.</p>
          <div className="mt-2">
            <AgentLogList entries={fixture.log} />
          </div>
        </section>
      </PopupBody>
    </PopupShell>
  );
}
