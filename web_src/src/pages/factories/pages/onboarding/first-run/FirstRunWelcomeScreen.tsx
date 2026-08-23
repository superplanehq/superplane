import { Button } from "@/components/ui/button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunWelcomeScreen({
  firstName,
  chrome,
  onGetStarted,
}: {
  firstName?: string;
  chrome?: FirstRunChrome;
  onGetStarted: () => void;
}) {
  const copy = FIRST_RUN_COPY.welcome;

  return (
    <FirstRunShell testId="first-run-welcome" chrome={chrome}>
      <FirstRunHeading
        greeting={firstName ? copy.greeting(firstName) : undefined}
        headline={copy.headline}
        size="display"
      >
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-10">
        <Button type="button" className="min-w-40" onClick={onGetStarted} data-testid="first-run-get-started">
          {copy.getStarted}
        </Button>
      </div>
    </FirstRunShell>
  );
}
