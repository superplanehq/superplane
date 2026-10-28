import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunCardReturnScreen({
  chrome,
  sphere,
  busy,
  error,
  onRetry,
  onBack,
}: {
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  busy: boolean;
  error?: string;
  onRetry: () => void;
  onBack: () => void;
}) {
  return (
    <FirstRunShell
      testId="first-run-card-return"
      chrome={{ ...chrome, onBack: undefined }}
      busy={busy}
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={error ? "Could not finish setup" : "Finishing setup…"}>
        <p role={error ? "alert" : "status"}>{error || "We are checking your plan and preparing your workspace."}</p>
      </FirstRunHeading>
      <LoadingButton className="mt-6 w-full" loading={busy} loadingText="Finishing setup…" onClick={onRetry}>
        Try again
      </LoadingButton>
      {error && (
        <Button variant="ghost" className="mt-3 w-full" onClick={onBack}>
          Back to plans
        </Button>
      )}
    </FirstRunShell>
  );
}
