import { Button } from "@/components/ui/button";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";

import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

const copy = OWNER_SETUP_COPY.fleet;

export function FleetManagerStep({ onContinue, onSkip }: { onContinue: () => void; onSkip: () => void }) {
  return (
    <div data-testid="owner-setup-fleet">
      <FirstRunHeading headline={copy.headline} size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4 text-left">
        <p className="text-[13px] leading-5 text-muted-foreground">{copy.helper}</p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Button type="button" className="min-w-40" onClick={onContinue}>
            {copy.continue}
          </Button>
          <Button type="button" variant="ghost" data-testid="owner-setup-fleet-skip" onClick={onSkip}>
            {copy.skip}
          </Button>
        </div>
      </div>
    </div>
  );
}
