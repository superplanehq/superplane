import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { BillingPlanFeatures } from "./BillingPlanFeatures";
import { billingCatalogRequest, type FeatureDefinition } from "./billingPlansApi";

export function Features() {
  const [features, setFeatures] = useState<FeatureDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const retryButton = useRef<HTMLButtonElement>(null);
  useReportPageReady(!loading, { failed: !!error });
  const load = async () => {
    try {
      setFeatures(await billingCatalogRequest<FeatureDefinition[]>("feature-definitions"));
      setError("");
    } catch {
      setError("Failed to load features");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  return (
    <div className="space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-xl font-semibold">Features</h1>
        <p className="text-sm text-muted-foreground">
          Manage feature access and view organizations with grants or plan inclusion.
        </p>
      </header>
      {error && (
        <div role="alert" className="flex items-center justify-between rounded-lg border p-4 text-sm">
          {error}
          <Button ref={retryButton} variant="outline" onClick={() => void load()}>
            Try again
          </Button>
        </div>
      )}
      {loading ? (
        <p role="status" className="text-sm text-muted-foreground">
          Loading features...
        </p>
      ) : (
        !error && (
          <BillingPlanFeatures features={features} onSaved={load} refreshErrorFocus={() => retryButton.current} />
        )
      )}
    </div>
  );
}
