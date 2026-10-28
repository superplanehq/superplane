import { useState, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { BillingPolarDiagnostics, OrganizationBillingPlan } from "./billingDiagnostics";

export type BillingDiagnosticsPanelProps = {
  enabled: boolean;
  plan: OrganizationBillingPlan | null;
  loading: boolean;
  checking: boolean;
  syncing: boolean;
  saving: boolean;
  error: string;
  syncError: string;
  onRetry: () => void;
  onCheck: () => void;
  onSync: () => void;
};

export function BillingDiagnosticsPanel({
  enabled,
  plan,
  loading,
  checking,
  syncing,
  saving,
  error,
  syncError,
  onRetry,
  onCheck,
  onSync,
}: BillingDiagnosticsPanelProps) {
  const [open, setOpen] = useState(false);
  const polar = plan?.polar;
  const busy = [loading, checking, syncing, saving].some(Boolean);
  return (
    <section
      aria-label="Billing diagnostics"
      aria-busy={busy}
      className="mt-5 rounded-md border border-gray-200 bg-white p-4 text-sm dark:border-gray-700 dark:bg-gray-900"
    >
      <h3>
        <Button
          type="button"
          variant="ghost"
          className="flex h-auto w-full items-center justify-between p-0 font-semibold"
          aria-expanded={open}
          onClick={() => setOpen((current) => !current)}
        >
          Billing diagnostics
          <ChevronDown aria-hidden="true" className={open ? "rotate-180" : ""} />
        </Button>
      </h3>
      {open && (
        <div className="mt-4 space-y-4">
          {!enabled ? (
            <p>Billing diagnostics are disabled on this installation.</p>
          ) : (
            <>
              {loading && <p role="status">Loading stored billing data...</p>}
              {error && (
                <div role="alert">
                  <p>{error}</p>
                  <Button variant="outline" size="sm" onClick={polar?.configured ? onCheck : onRetry} disabled={busy}>
                    Try again
                  </Button>
                </div>
              )}
              {!plan && !loading && !error && <p>Stored billing data is unavailable.</p>}
              {plan && (
                <>
                  <StoredBillingData plan={plan} />
                  <PolarBillingData
                    polar={polar}
                    busy={busy}
                    checking={checking}
                    syncing={syncing}
                    syncError={syncError}
                    onCheck={onCheck}
                    onSync={onSync}
                  />
                  <RecordedBillingEvidence plan={plan} />
                </>
              )}
            </>
          )}
        </div>
      )}
    </section>
  );
}

function StoredBillingData({ plan }: { plan: OrganizationBillingPlan }) {
  return (
    <div className="space-y-2">
      <h4 className="font-medium">Stored billing data</h4>
      {plan.record_exists === false ? (
        <p>No billing-plan record</p>
      ) : (
        <dl className="grid gap-2 sm:grid-cols-2">
          <Datum label="Stored plan">{plan.plan}</Datum>
          <Datum label="Catalog plan">
            {plan.catalog_name || "Unknown"} ({plan.catalog_key || "Unknown"})
          </Datum>
          <Datum label="Source">{plan.plan_source || "Unknown"}</Datum>
          <Datum label="Updated">
            <Timestamp value={plan.updated_at} />
          </Datum>
          <Datum label="Trial started">
            <Timestamp value={plan.trial_started_at} />
          </Datum>
          <Datum label="Stored trial ends">
            <Timestamp value={plan.trial_ends_at} />
          </Datum>
          <Datum label="Subscription period starts">
            <Timestamp value={plan.current_period_start} />
          </Datum>
          <Datum label="Subscription period ends">
            <Timestamp value={plan.current_period_end} />
          </Datum>
        </dl>
      )}
      <dl className="grid gap-2 sm:grid-cols-2">
        <Datum label="Stored Polar customer ID">{plan.polar_customer_id || "Not stored"}</Datum>
        <Datum label="Stored Polar subscription ID">{plan.polar_subscription_id || "Not stored"}</Datum>
        <Datum label="Stored subscription status">{plan.polar_subscription_status || "Not stored"}</Datum>
      </dl>
      <div className="flex flex-wrap gap-3">
        <DiagnosticLink href={plan.polar_customer_url}>View stored customer in Polar</DiagnosticLink>
        <DiagnosticLink href={plan.polar_customer_events_url}>View stored customer events in Polar</DiagnosticLink>
        <DiagnosticLink href={plan.polar_subscription_url}>View stored subscription in Polar</DiagnosticLink>
      </div>
    </div>
  );
}

function PolarBillingData({
  polar,
  busy,
  checking,
  syncing,
  syncError,
  onCheck,
  onSync,
}: {
  polar: OrganizationBillingPlan["polar"];
  busy: boolean;
  checking: boolean;
  syncing: boolean;
  syncError: string;
  onCheck: () => void;
  onSync: () => void;
}) {
  const canSync = polar?.configured && polar.customer_status === "found" && polar.card_status === "usable";
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h4 className="font-medium">Polar</h4>
        <Button variant="outline" size="sm" onClick={onCheck} disabled={!polar?.configured || busy}>
          {checking ? "Checking Polar..." : "Check Polar"}
        </Button>
      </div>
      {!polar?.configured ? (
        <p>Polar is not configured. Local plan assignment is available.</p>
      ) : (
        <>
          <p>Environment: {polar.environment}</p>
          <div className="space-y-1">
            <Button variant="outline" size="sm" onClick={onSync} disabled={!canSync || busy}>
              {syncing ? "Syncing customer..." : "Sync customer from Polar"}
            </Button>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Verify the saved card in Polar and update the customer ID in SuperPlane.
            </p>
          </div>
          <PolarCheckResults polar={polar} />
        </>
      )}
      {syncError && <p role="alert">{syncError}</p>}
    </div>
  );
}

function PolarCheckResults({ polar }: { polar: BillingPolarDiagnostics }) {
  if (!polar.checked_at)
    return <p>Not checked. Select Check Polar to load current billing data and enable plan changes.</p>;

  return (
    <>
      <p>
        Checked: <Timestamp value={polar.checked_at} />
      </p>
      <dl className="grid gap-2 sm:grid-cols-2">
        <Datum label="External customer ID">{polar.external_id}</Datum>
        <Datum label="Customer lookup">{statusLabel(polar.customer_status)}</Datum>
        <Datum label="Live customer ID">{polar.customer_id || "Unknown"}</Datum>
        <Datum label="Usable saved card">{cardLabel(polar.card_status)}</Datum>
        <Datum label="Plan editing verification">{statusLabel(polar.subscription_verification)}</Datum>
      </dl>
      {polar.customer_id_mismatch && <p role="alert">The live customer ID differs from the stored ID.</p>}
      {polar.customer_error && <p role="alert">{polar.customer_error}</p>}
      {polar.card_error && <p role="alert">{polar.card_error}</p>}
      {polar.subscription_error && <p role="alert">{polar.subscription_error}</p>}
      <div className="flex flex-wrap gap-3">
        <DiagnosticLink href={polar.customer_url}>View live customer in Polar</DiagnosticLink>
        <DiagnosticLink href={polar.customer_events_url}>View live customer events in Polar</DiagnosticLink>
      </div>
      <p className="font-medium">Live subscriptions</p>
      {polar.subscriptions.length === 0 && (
        <p>
          {polar.subscription_verification === "failed"
            ? "Unknown. Subscription check failed."
            : "No subscriptions returned."}
        </p>
      )}
      <ul className="space-y-1">
        {polar.subscriptions.map((subscription) => (
          <li key={subscription.id}>
            <DiagnosticLink href={subscription.url}>{subscription.id}</DiagnosticLink> — {subscription.status}
          </li>
        ))}
      </ul>
    </>
  );
}

function RecordedBillingEvidence({ plan }: { plan: OrganizationBillingPlan }) {
  return (
    <div className="space-y-2">
      <h4 className="font-medium">Recorded evidence</h4>
      <p>Historical card changes: Unknown. SuperPlane does not record card additions or removals.</p>
      <p className="font-medium">Workspace onboarding</p>
      {!plan.workspaces?.length && <p>No workspaces recorded.</p>}
      <ul className="space-y-1">
        {plan.workspaces?.map((workspace) => (
          <li key={workspace.id}>
            {workspace.name} ({workspace.id}):{" "}
            {workspace.onboarding_completed_at ? (
              <Timestamp value={workspace.onboarding_completed_at} />
            ) : (
              "Not completed"
            )}
          </li>
        ))}
      </ul>
      <p className="font-medium">Latest checkout attempts (up to 20)</p>
      {!plan.checkout_attempts?.length && <p>No checkout attempts recorded.</p>}
      <ul className="space-y-3">
        {plan.checkout_attempts?.map((attempt) => (
          <li key={attempt.id} className="rounded border border-gray-200 p-3 dark:border-gray-700">
            <dl className="grid gap-2 sm:grid-cols-2">
              <Datum label="Selected plan">{attempt.plan_key}</Datum>
              <Datum label="Local state">{attempt.state}</Datum>
              <Datum label="Polar checkout ID">{attempt.polar_checkout_id || "Not recorded"}</Datum>
              <Datum label="Recorded Polar status">{attempt.polar_status || "Not recorded"}</Datum>
              <Datum label="Created">
                <Timestamp value={attempt.created_at} />
              </Datum>
              <Datum label="Updated">
                <Timestamp value={attempt.updated_at} />
              </Datum>
              <Datum label="Expires">
                <Timestamp value={attempt.expires_at} />
              </Datum>
              <Datum label="Resolved">
                <Timestamp value={attempt.resolved_at} />
              </Datum>
              <Datum label="Failure information">{attempt.last_error || "Not recorded"}</Datum>
            </dl>
          </li>
        ))}
      </ul>
    </div>
  );
}

function Datum({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-gray-500 dark:text-gray-400">{label}</dt>
      <dd className="break-words">{children}</dd>
    </div>
  );
}

function Timestamp({ value }: { value?: string | null }) {
  if (!value) return <>Not recorded</>;
  return (
    <time dateTime={value}>
      {new Date(value).toISOString().replace("T", " ").replace(".000Z", " UTC").replace("Z", " UTC")}
    </time>
  );
}

function DiagnosticLink({ href, children }: { href?: string; children: ReactNode }) {
  if (!href) return null;
  return (
    <Button variant="link" size="sm" className="h-auto p-0 text-xs" asChild>
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    </Button>
  );
}

function statusLabel(status: string) {
  return (
    (
      {
        found: "Found",
        absent: "Absent",
        failed: "Failed",
        unknown: "Unknown",
        active: "Locked: live subscription",
        none: "No live subscription",
        not_checked: "Not checked",
        not_configured: "Not configured",
      } as Record<string, string>
    )[status] ?? status
  );
}

function cardLabel(status: string) {
  if (status === "usable") return "Usable card found";
  if (status === "absent") return "No usable card (absent or expired)";
  return statusLabel(status);
}
