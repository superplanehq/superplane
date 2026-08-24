import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderCheckCard } from "../../WorkOrderChecksSection";
import { DispatchTimelineItem } from "../../timeline/DispatchTimelineItem";
import { OwnerTimeCostRow, PopupBody, PopupHeader, PopupShell, SectionTitle, WaitingNotes } from "./popupShared";
import { buildPopupDispatchEvent, type PopupFixture } from "./workOrderPopupMocks";

/**
 * Job report. Pattern: Laravel Cloud / Railway deploy details.
 *
 * First screen: next step, scores, then the activity log.
 * Artifacts hang on the step that produced them. Markdown opens a
 * second popup. Pull requests and branches open in a new tab.
 */
export function ConceptJobPopup({
  fixture,
  organizationId = FACTORIES_ORGANIZATION_ID,
  factoryKey = PRIMARY_FACTORY_KEY,
  onClose,
  fixed = false,
}: {
  fixture: PopupFixture;
  organizationId?: string;
  factoryKey?: string;
  onClose?: () => void;
  fixed?: boolean;
}) {
  const dispatch = buildPopupDispatchEvent(fixture);

  return (
    <PopupShell testId="work-order-popup-job" fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={fixture.title} onClose={onClose}>
        <OwnerTimeCostRow fixture={fixture} />
      </PopupHeader>
      <PopupBody>
        <div className="flex flex-col gap-8">
          <WaitingNotes notes={fixture.waitingNotes} />

          {fixture.checks.length > 0 ? (
            <section>
              <SectionTitle>Scores</SectionTitle>
              <div className="mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                {fixture.checks.map((check) => (
                  <WorkOrderCheckCard key={check.id} check={check} />
                ))}
              </div>
            </section>
          ) : null}

          <section>
            <div className="flex items-baseline justify-between gap-3">
              <SectionTitle>Log</SectionTitle>
              <span className="text-[12px] text-muted-foreground">{fixture.elapsed}</span>
            </div>
            <div className="mt-2">
              {dispatch ? (
                <ul className="relative space-y-4">
                  <span
                    className="pointer-events-none absolute top-3 bottom-3 left-[11px] w-px bg-border"
                    aria-hidden
                  />
                  <DispatchTimelineItem
                    event={dispatch}
                    organizationId={organizationId}
                    factoryKey={factoryKey}
                    isLatestDispatch
                  />
                </ul>
              ) : (
                <p className="text-[13px] text-muted-foreground">No steps yet.</p>
              )}
            </div>
          </section>
        </div>
      </PopupBody>
    </PopupShell>
  );
}
