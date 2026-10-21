import { useCallback, useEffect, useState } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { billingCatalogRequest, type FeatureDefinition, type OrganizationFeatures } from "./billingPlansApi";

export function OrganizationFeatureGrants({ orgId }: { orgId: string }) {
  const [definitions, setDefinitions] = useState<FeatureDefinition[]>([]);
  const [state, setState] = useState<OrganizationFeatures | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    try {
      const [definitions, state] = await Promise.all([
        billingCatalogRequest<FeatureDefinition[]>("feature-definitions"),
        billingCatalogRequest<OrganizationFeatures>(`organizations/${orgId}/features`),
      ]);
      setDefinitions(definitions);
      setState(state);
      setError("");
    } catch {
      setError("Failed to load feature access. Try again.");
    }
  }, [orgId]);
  useEffect(() => {
    void load();
  }, [load]);
  const save = async (key: string, enabled: boolean) => {
    setBusy(true);
    try {
      setState(
        await billingCatalogRequest<OrganizationFeatures>(`organizations/${orgId}/features`, "PUT", {
          feature_key: key,
          enabled,
        }),
      );
      setError("");
    } catch {
      setError("Failed to save feature grant. Try again.");
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="mt-6 space-y-3">
      <h3 className="font-semibold">Feature grants</h3>
      <p>Grants add access. Removing a grant keeps access from the plan or an ungated feature.</p>
      {error && (
        <div role="alert">
          {error} <Button onClick={() => void load()}>Try again</Button>
        </div>
      )}
      {state &&
        definitions.map((feature) => (
          <div key={feature.key} className="flex items-center gap-3">
            <Label className="flex items-center gap-2">
              <Checkbox
                disabled={busy}
                checked={state.grants.some((grant) => grant.feature_key === feature.key && grant.enabled)}
                onChange={(event) => void save(feature.key, event.target.checked)}
              />
              {feature.label}
            </Label>
            <span>
              {state.effective.features[feature.key] ? "Enabled" : "Disabled"}
              {!feature.gated && " · Available to all organizations"}
            </span>
          </div>
        ))}
    </section>
  );
}
