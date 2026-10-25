import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useCallback, useEffect, useState } from "react";
import { formatDateTime } from "./formatDate";

type BuiltInAutomation = {
  id: string;
  name: string;
  last_reset_at?: string;
  last_reset_by?: string;
};

type ResetFailure = {
  organization_id: string;
  organization_name: string;
  error: string;
};

type ResetResult = {
  reset?: number;
  organizations?: number;
  failed?: ResetFailure[];
  last_reset_at?: string;
  last_reset_by?: string;
};

const LOAD_ERROR = "Could not load built-in automations.";
const RESET_ERROR = "Could not reset this automation.";
const NEVER_RESET = "—";

const AUTOMATION_GROUPS = [
  { id: "intake", label: "Intake" },
  { id: "impl", label: "Implementation" },
  { id: "verify", label: "Verify" },
  { id: "done", label: "Done" },
] as const;

type AutomationGroupId = (typeof AUTOMATION_GROUPS)[number]["id"];

const IMPL_AUTOMATION_IDS = new Set(["line-implementation"]);
const VERIFY_AUTOMATION_IDS = new Set(["risk-score", "visual-evidence"]);
const DONE_AUTOMATION_IDS = new Set(["pr-closure"]);

function automationGroup(id: string): AutomationGroupId {
  if (IMPL_AUTOMATION_IDS.has(id)) {
    return "impl";
  }
  if (DONE_AUTOMATION_IDS.has(id)) {
    return "done";
  }
  if (VERIFY_AUTOMATION_IDS.has(id) || id.startsWith("pr-feedback:")) {
    return "verify";
  }
  return "intake";
}

function groupedAutomations(automations: BuiltInAutomation[]) {
  return AUTOMATION_GROUPS.map((group) => ({
    ...group,
    automations: automations.filter((automation) => automationGroup(automation.id) === group.id),
  })).filter((group) => group.automations.length > 0);
}

export function ResetBuiltInAutomation() {
  const [automations, setAutomations] = useState<BuiltInAutomation[]>([]);
  const [selected, setSelected] = useState<BuiltInAutomation | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [failures, setFailures] = useState<ResetFailure[]>([]);
  const [error, setError] = useState<string | null>(null);

  const loadAutomations = useCallback(async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const response = await fetch("/admin/api/built-in-automations", { credentials: "include" });
      if (!response.ok) {
        setLoadError(true);
        setLoading(false);
        return;
      }
      const body = (await response.json()) as { automations?: BuiltInAutomation[] };
      setAutomations(body.automations ?? []);
      setLoading(false);
    } catch {
      setLoadError(true);
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadAutomations();
  }, [loadAutomations]);

  const reset = async () => {
    if (!selected) {
      return;
    }
    const automation = selected;
    setPending(true);
    setError(null);
    setMessage(null);
    setFailures([]);
    try {
      const data = await requestAutomationReset(automation.id);
      const failed = data.failed ?? [];
      setOpen(false);
      setMessage(resetResultMessage(automation.name, data.reset ?? 0, data.organizations ?? 0, failed.length));
      setFailures(failed);
      recordReset(setAutomations, automation.id, data);
    } catch (cause) {
      setError(cause instanceof ResetRequestError ? cause.message : RESET_ERROR);
    } finally {
      setPending(false);
    }
  };

  return (
    <section className="py-6" aria-labelledby="reset-built-in-automation-heading">
      <ResetAutomationHeading />
      <CatalogNotice
        loading={loading}
        loadError={loadError}
        empty={!loading && !loadError && automations.length === 0}
        onRetry={() => void loadAutomations()}
      />
      {!loading && !loadError && automations.length > 0 ? (
        <AutomationCatalogTable
          automations={automations}
          pending={pending}
          selectedId={selected?.id ?? null}
          onReset={(automation) => {
            setError(null);
            setSelected(automation);
            setOpen(true);
          }}
        />
      ) : null}
      {message ? <Text className="mt-3 text-sm text-gray-500 dark:text-gray-400">{message}</Text> : null}
      <FailureList failures={failures} />
      <ResetConfirmDialog
        open={open}
        pending={pending}
        selected={selected}
        error={error}
        onClose={() => setOpen(false)}
        onConfirm={() => void reset()}
      />
    </section>
  );
}

function ResetAutomationHeading() {
  return (
    <div className="max-w-2xl">
      <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
        Built-in automations
      </p>
      <h2
        id="reset-built-in-automation-heading"
        className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100"
      >
        Reset one automation
      </h2>
      <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Replace one built-in automation in every organization and workspace. User settings stay unchanged.
      </Text>
    </div>
  );
}

function CatalogNotice({
  loading,
  loadError,
  empty,
  onRetry,
}: {
  loading: boolean;
  loadError: boolean;
  empty: boolean;
  onRetry: () => void;
}) {
  if (loadError) {
    return (
      <div className="mt-5 flex items-center gap-3" role="alert">
        <Text className="text-sm text-red-600 dark:text-red-400">{LOAD_ERROR}</Text>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      </div>
    );
  }
  if (loading) {
    return <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading automations...</Text>;
  }
  if (empty) {
    return (
      <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">No built-in automations are available.</Text>
    );
  }
  return null;
}

function AutomationCatalogTable({
  automations,
  pending,
  selectedId,
  onReset,
}: {
  automations: BuiltInAutomation[];
  pending: boolean;
  selectedId: string | null;
  onReset: (automation: BuiltInAutomation) => void;
}) {
  return (
    <div className="mt-5 overflow-hidden rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Automation</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Last reset by</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Last reset at</th>
            <th className="px-4 py-2.5 text-right">
              <span className="sr-only">Reset</span>
            </th>
          </tr>
        </thead>
        {groupedAutomations(automations).map((group) => (
          <tbody key={group.id} aria-label={group.label}>
            <tr className="border-b border-slate-100 bg-slate-50 dark:border-gray-700/70 dark:bg-gray-800/60">
              <th
                colSpan={4}
                scope="colgroup"
                className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400"
              >
                {group.label}
              </th>
            </tr>
            {group.automations.map((automation) => (
              <AutomationCatalogRow
                key={automation.id}
                automation={automation}
                pending={pending}
                resetting={pending && selectedId === automation.id}
                onReset={onReset}
              />
            ))}
          </tbody>
        ))}
      </table>
    </div>
  );
}

function AutomationCatalogRow({
  automation,
  pending,
  resetting,
  onReset,
}: {
  automation: BuiltInAutomation;
  pending: boolean;
  resetting: boolean;
  onReset: (automation: BuiltInAutomation) => void;
}) {
  return (
    <tr className="border-b border-slate-50 last:border-0 dark:border-gray-800/70">
      <td className="px-4 py-2.5 font-medium text-gray-800 dark:text-gray-100">{automation.name}</td>
      <td className="px-4 py-2.5 text-gray-500 dark:text-gray-400">{automation.last_reset_by || NEVER_RESET}</td>
      <td className="px-4 py-2.5 whitespace-nowrap text-gray-500 dark:text-gray-400">
        {formatDateTime(automation.last_reset_at)}
      </td>
      <td className="px-4 py-2.5 text-right">
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label={`Reset ${automation.name}`}
          disabled={pending}
          onClick={() => onReset(automation)}
        >
          {resetting ? "Resetting..." : "Reset"}
        </Button>
      </td>
    </tr>
  );
}

function FailureList({ failures }: { failures: ResetFailure[] }) {
  if (failures.length === 0) {
    return null;
  }
  return (
    <ul className="mt-2 space-y-1" role="alert">
      {failures.map((failure) => (
        <li key={failure.organization_id} className="text-sm text-red-600 dark:text-red-400">
          {failure.organization_name}: {failure.error}
        </li>
      ))}
    </ul>
  );
}

function ResetConfirmDialog({
  open,
  pending,
  selected,
  error,
  onClose,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  selected: BuiltInAutomation | null;
  error: string | null;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const name = selected?.name ?? "this automation";
  return (
    <Dialog
      open={open}
      onClose={() => (pending ? undefined : onClose())}
      size="md"
      role="dialog"
      aria-label={selected ? `Reset ${selected.name}` : "Reset automation"}
      aria-modal="true"
    >
      <DialogTitle>{selected ? `Reset ${selected.name}?` : "Reset automation?"}</DialogTitle>
      <DialogDescription>
        <p className="text-sm leading-6 text-gray-800 dark:text-gray-200">
          This replaces {name} in every organization and workspace with the current template.
        </p>
        {error ? (
          <p role="alert" className="mt-2 text-red-600 dark:text-red-400">
            {error}
          </p>
        ) : null}
      </DialogDescription>
      <DialogActions>
        <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
          Cancel
        </Button>
        <Button type="button" variant="destructive" disabled={pending || !selected} onClick={onConfirm}>
          {pending ? "Resetting..." : selected ? `Reset ${selected.name}` : "Reset automation"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

class ResetRequestError extends Error {}

async function requestAutomationReset(id: string): Promise<ResetResult> {
  const response = await fetch("/admin/api/built-in-automations/reset", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
  if (!response.ok) {
    const details = response.status === 400 ? (await response.text()).trim() : "";
    throw new ResetRequestError(details || RESET_ERROR);
  }
  return (await response.json()) as ResetResult;
}

function recordReset(
  setAutomations: (update: (current: BuiltInAutomation[]) => BuiltInAutomation[]) => void,
  id: string,
  data: ResetResult,
) {
  if (!data.last_reset_at) {
    return;
  }
  setAutomations((current) =>
    current.map((item) =>
      item.id === id ? { ...item, last_reset_at: data.last_reset_at, last_reset_by: data.last_reset_by } : item,
    ),
  );
}

function resetResultMessage(name: string, reset: number, organizations: number, failed: number): string {
  const resetText =
    reset === 0
      ? `No ${name} automations were reset.`
      : `Reset ${countText(reset, `${name} automation`, `${name} automations`)} in ${countText(organizations, "organization", "organizations")}.`;
  if (failed === 0) {
    return resetText;
  }
  const failureText = failed === 1 ? "1 organization was not changed." : `${failed} organizations were not changed.`;
  return `${resetText} ${failureText}`;
}

function countText(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`;
}
