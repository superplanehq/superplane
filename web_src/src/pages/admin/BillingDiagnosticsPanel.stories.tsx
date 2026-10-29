import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { BillingDiagnosticsPanel } from "./BillingDiagnosticsPanel";
import type { OrganizationBillingPlan } from "./useOrgLLMCredit";

const plan: OrganizationBillingPlan = {
  record_exists: true,
  plan: "trial",
  catalog_key: "trial",
  catalog_name: "Trial",
  plan_source: "system",
  updated_at: "2026-10-28T09:00:00Z",
  trial_started_at: "2026-10-21T09:00:00Z",
  trial_ends_at: "2026-11-04T09:00:00Z",
  current_period_end: null,
  polar_managed: false,
  polar_subscription_status: "",
  polar_subscription_verification: "not_checked",
  polar_customer_id: "cust_stored",
  polar_customer_url: "https://sandbox.polar.sh/dashboard/superplane/customers/cust_stored",
  workspaces: [
    { id: "workspace-1", name: "Engineering", onboarding_completed_at: "2026-10-21T10:00:00Z" },
    { id: "workspace-2", name: "Support", onboarding_completed_at: null },
  ],
  checkout_attempts: [
    {
      id: "attempt-1",
      plan_key: "team",
      state: "failed",
      polar_checkout_id: "checkout-1",
      polar_status: "failed",
      last_error: "Payment declined",
      created_at: "2026-10-27T11:00:00Z",
      updated_at: "2026-10-27T11:02:00Z",
      expires_at: null,
      resolved_at: "2026-10-27T11:02:00Z",
    },
  ],
  polar: {
    configured: true,
    environment: "sandbox",
    external_id: "organization-1",
    checked_at: null,
    customer_status: "not_checked",
    customer_id: "",
    customer_id_mismatch: false,
    card_status: "not_checked",
    subscription_verification: "not_checked",
    subscriptions: [],
  },
};
const checked = {
  ...plan.polar!,
  checked_at: "2026-10-28T12:00:00Z",
  customer_status: "found",
  customer_id: "cust_live",
  customer_id_mismatch: true,
  card_status: "usable",
  subscription_verification: "active",
  subscriptions: [
    {
      id: "sub_live",
      status: "paused",
      url: "https://sandbox.polar.sh/dashboard/superplane/sales/subscriptions/sub_live",
    },
  ],
};
function logDiagnosticAction(action: string) {
  console.log(action);
}
function InteractiveDiagnostics(args: React.ComponentProps<typeof BillingDiagnosticsPanel>) {
  const [currentPlan, setCurrentPlan] = useState(args.plan);
  const [error, setError] = useState(args.error);
  const [syncError, setSyncError] = useState(args.syncError);
  return (
    <BillingDiagnosticsPanel
      {...args}
      plan={currentPlan}
      error={error}
      syncError={syncError}
      onCheck={() => {
        logDiagnosticAction("Check Polar");
        if (currentPlan) setCurrentPlan({ ...currentPlan, polar: checked });
        setError("");
        setSyncError("");
      }}
      onRetry={() => {
        logDiagnosticAction("Retry stored billing data");
        setCurrentPlan(plan);
        setError("");
        setSyncError("");
      }}
      onSync={() => {
        logDiagnosticAction("Sync customer from Polar");
        if (currentPlan?.polar)
          setCurrentPlan({
            ...currentPlan,
            polar_customer_id: currentPlan.polar.customer_id,
            polar: { ...currentPlan.polar, customer_id_mismatch: false },
          });
        setSyncError("");
      }}
    />
  );
}

const meta = {
  title: "Admin/BillingDiagnosticsPanel",
  component: BillingDiagnosticsPanel,
  render: (args) => <InteractiveDiagnostics {...args} />,
  args: {
    enabled: true,
    plan,
    loading: false,
    checking: false,
    syncing: false,
    saving: false,
    error: "",
    syncError: "",
    onCheck: () => logDiagnosticAction("Check Polar"),
    onRetry: () => logDiagnosticAction("Retry stored billing data"),
    onSync: () => logDiagnosticAction("Sync customer from Polar"),
  },
} satisfies Meta<typeof BillingDiagnosticsPanel>;
export default meta;
type Story = StoryObj<typeof meta>;
export const NotChecked: Story = {};
export const Loading: Story = { args: { plan: null, loading: true } };
export const Checking: Story = { args: { checking: true } };
export const Disabled: Story = { args: { enabled: false, plan: null } };
export const NotConfigured: Story = { args: { plan: { ...plan, polar: { ...plan.polar!, configured: false } } } };
export const MissingRecord: Story = {
  args: { plan: { ...plan, record_exists: false, plan: "none", checkout_attempts: [] } },
};
export const SubscriptionLocked: Story = { args: { plan: { ...plan, polar: checked } } };
export const SyncAvailable: Story = {
  args: {
    plan: {
      ...plan,
      polar_subscription_verification: "none",
      polar: { ...checked, subscription_verification: "none", subscriptions: [] },
    },
  },
};
export const Syncing: Story = { args: { plan: { ...plan, polar: checked }, syncing: true } };
export const SyncComplete: Story = {
  args: {
    plan: { ...plan, polar_customer_id: checked.customer_id, polar: { ...checked, customer_id_mismatch: false } },
  },
};
export const SyncFailed: Story = {
  args: {
    plan: { ...plan, polar: checked },
    syncError: "Could not sync the Polar customer. Try again.",
  },
};
export const NoUsableCard: Story = { args: { plan: { ...plan, polar: { ...checked, card_status: "absent" } } } };
export const PartialFailure: Story = {
  args: {
    plan: {
      ...plan,
      polar: {
        ...checked,
        card_status: "failed",
        card_error: "Could not verify saved cards. Check Polar and try again.",
      },
    },
  },
};
export const CustomerFailure: Story = {
  args: {
    plan: {
      ...plan,
      polar: {
        ...checked,
        customer_status: "failed",
        customer_id: "",
        customer_id_mismatch: false,
        customer_error: "Could not verify the Polar customer. Check Polar and try again.",
        card_status: "unknown",
      },
    },
  },
};
export const LoadFailed: Story = { args: { plan: null, error: "Could not load billing diagnostics. Try again." } };
export const CheckFailed: Story = { args: { error: "Could not load billing diagnostics. Try again." } };
