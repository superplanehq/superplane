import { useEffect, useRef, useState } from "react";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { billingCatalogRequest, type FeatureDefinition, type OrganizationFeatures } from "./billingPlansApi";

export function OrganizationFeatureGrants({ orgId }: { orgId: string }) {
  const [definitions, setDefinitions] = useState<FeatureDefinition[]>([]);
  const [state, setState] = useState<OrganizationFeatures | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);
  const activeOrg = useRef(orgId);
  const [loadedOrg, setLoadedOrg] = useState(orgId);
  if (loadedOrg !== orgId) {
    // Reset grants before they render for a different organization, so stale
    // controls cannot change the organization that is selected now.
    setLoadedOrg(orgId);
    setDefinitions([]);
    setState(null);
    setError("");
    setBusy(false);
  }
  useEffect(() => {
    activeOrg.current = orgId;
    let cancelled = false;
    const load = async () => {
      try {
        const [definitions, state] = await Promise.all([
          billingCatalogRequest<FeatureDefinition[]>("feature-definitions"),
          billingCatalogRequest<OrganizationFeatures>(`organizations/${orgId}/features`),
        ]);
        if (cancelled) return;
        setDefinitions(definitions);
        setState(state);
        setError("");
      } catch {
        if (!cancelled) setError("Failed to load feature access. Try again.");
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [orgId, reloadToken]);
  const save = async (key: string, enabled: boolean) => {
    const targetOrg = orgId;
    setBusy(true);
    try {
      const updated = await billingCatalogRequest<OrganizationFeatures>(`organizations/${targetOrg}/features`, "PUT", {
        feature_key: key,
        enabled,
      });
      if (activeOrg.current !== targetOrg) return;
      setState(updated);
      setError("");
    } catch {
      if (activeOrg.current === targetOrg) setError("Failed to save feature grant. Try again.");
    } finally {
      if (activeOrg.current === targetOrg) setBusy(false);
    }
  };
  return (
    <section className="mt-6 space-y-3">
      <h3 className="font-semibold">Features</h3>
      <p className="text-sm text-muted-foreground">
        Manual grants add access. Removing a grant keeps access included in the plan or available to all organizations.
      </p>
      {error && (
        <div role="alert">
          {error} <Button onClick={() => setReloadToken((token) => token + 1)}>Try again</Button>
        </div>
      )}
      {state && (
        <Table>
          <caption className="sr-only">Organization features</caption>
          <TableHead>
            <TableRow>
              <TableHeader>Feature</TableHeader>
              <TableHeader>Access</TableHeader>
              <TableHeader>Source</TableHeader>
              <TableHeader>
                <span className="sr-only">Actions</span>
              </TableHeader>
            </TableRow>
          </TableHead>
          <TableBody>
            {definitions.length === 0 && (
              <TableRow>
                <TableCell colSpan={4}>No features defined.</TableCell>
              </TableRow>
            )}
            {definitions.map((feature) => {
              const granted = state.grants.some((grant) => grant.feature_key === feature.key && grant.enabled);
              const enabled = state.effective.features[feature.key];
              const source = granted
                ? "Manual grant"
                : !feature.gated
                  ? "All organizations"
                  : enabled
                    ? "Plan"
                    : "No access";
              return (
                <TableRow key={feature.key}>
                  <TableCell className="font-medium">{feature.label}</TableCell>
                  <TableCell>{enabled ? "Enabled" : "Disabled"}</TableCell>
                  <TableCell>{source}</TableCell>
                  <TableCell className="text-right">
                    {(granted || (feature.gated && !enabled)) && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={busy}
                        aria-label={`${granted ? "Remove grant for" : "Add grant for"} ${feature.label}`}
                        onClick={() => void save(feature.key, !granted)}
                      >
                        {granted ? "Remove grant" : "Add grant"}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}
    </section>
  );
}
