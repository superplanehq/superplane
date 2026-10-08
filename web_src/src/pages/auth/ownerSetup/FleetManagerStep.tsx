import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { CopyButton } from "@/ui/CopyButton";
import { useCallback, useEffect, useState } from "react";

import { ErrorBanner } from "./ErrorBanner";
import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

const copy = OWNER_SETUP_COPY.fleet;

type FleetManagerConfig = {
  yaml: string;
};

export function FleetManagerStep({ onContinue, onSkip }: { onContinue: () => void; onSkip: () => void }) {
  const [yaml, setYaml] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch("/admin/api/installation/first-run/fleet-manager", {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(copy.error);
      }
      const body = (await response.json()) as FleetManagerConfig;
      if (!body.yaml) {
        throw new Error(copy.error);
      }
      setYaml(body.yaml);
    } catch {
      setYaml("");
      setError(copy.error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div data-testid="owner-setup-fleet">
      <FirstRunHeading headline={copy.headline} size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4 text-left">
        <ErrorBanner message={error} />
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {copy.loading}
          </p>
        ) : yaml ? (
          <>
            <p className="text-[13px] text-muted-foreground">{copy.helper}</p>
            <Textarea
              readOnly
              value={yaml}
              className="min-h-56 font-mono text-[12px]"
              data-testid="owner-setup-fleet-yaml"
            />
            <CopyButton variant="button" text={yaml} copiedLabel={copy.copied} data-testid="owner-setup-fleet-copy">
              {copy.copy}
            </CopyButton>
          </>
        ) : (
          <Button type="button" variant="outline" onClick={() => void load()}>
            {copy.retry}
          </Button>
        )}
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Button type="button" className="min-w-40" disabled={loading || !yaml} onClick={onContinue}>
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
