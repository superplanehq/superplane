import { Button } from "@/components/ui/button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import { FirstRunTicketList } from "./FirstRunTicketList";
import type { FirstRunChrome, FirstRunScoredTicket } from "./firstRunTypes";

export function FirstRunWelcomeScreen({
  firstName,
  previewTickets,
  chrome,
  onGetStarted,
}: {
  firstName?: string;
  previewTickets: FirstRunScoredTicket[];
  chrome?: FirstRunChrome;
  onGetStarted: () => void;
}) {
  const copy = FIRST_RUN_COPY.welcome;

  return (
    <FirstRunShell testId="first-run-welcome" chrome={chrome} width="wide">
      <FirstRunHeading
        greeting={firstName ? copy.greeting(firstName) : undefined}
        headline={copy.headline}
        size="display"
      >
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <figure className="mt-10">
        <FirstRunTicketList tickets={previewTickets} />
        <figcaption className="mt-3 text-[12px] text-muted-foreground">{copy.previewCaption}</figcaption>
      </figure>

      <div className="mt-10">
        <Button type="button" className="min-w-40" onClick={onGetStarted} data-testid="first-run-get-started">
          {copy.getStarted}
        </Button>
      </div>
    </FirstRunShell>
  );
}
