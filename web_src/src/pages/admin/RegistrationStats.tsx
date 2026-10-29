import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { Heading } from "@/components/Heading/heading";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";
import AdminPagination from "./AdminPagination";
import { registrationPeriod, type RegistrationPreset, type RegistrationPeriod } from "./registrationPeriod";

const PAGE_SIZE = 5;
const presets: { value: RegistrationPreset | "custom"; label: string }[] = [
  { value: "today", label: "Today" },
  { value: "week", label: "This week" },
  { value: "month", label: "This month" },
  { value: "7days", label: "Last 7 days" },
  { value: "30days", label: "Last 30 days" },
  { value: "custom", label: "Custom" },
];

export type RegistrationReport = {
  items: { id: string; name: string; owner_email: string; plan: string; available_credit_cents: number }[];
  total: number;
  trial_granted_cents: number;
  trial_spent_cents: number;
  plan_activations: { team: number; starter: number; pay_as_you_go: number };
  // ISO timestamp. Activations before this time are not tracked.
  plan_activation_tracking_started_at: string;
};

type RegistrationResponse = Omit<RegistrationReport, "trial_granted_cents" | "trial_spent_cents"> &
  Partial<Pick<RegistrationReport, "trial_granted_cents" | "trial_spent_cents">>;

function dateInputValue(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

export function RegistrationStats() {
  const [query, setQuery] = useState(() => ({ ...registrationPeriod("today"), offset: 0 }));
  const [report, setReport] = useState<RegistrationReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  // Only the first page returns credit totals. Keep them while the user pages.
  const totals = useRef<{ key: string; granted: number; spent: number } | null>(null);
  useReportPageReady(!loading, { failed: error });

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError(false);
    const params = new URLSearchParams({ start: query.start, end: query.end, offset: String(query.offset) });
    void fetch(`/admin/api/stats?${params}`, { credentials: "include", signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error("Failed to load statistics");
        const result: RegistrationResponse = await response.json();
        const key = `${query.start}|${query.end}`;
        if (result.trial_granted_cents !== undefined && result.trial_spent_cents !== undefined) {
          totals.current = { key, granted: result.trial_granted_cents, spent: result.trial_spent_cents };
        }
        const known = totals.current?.key === key ? totals.current : null;
        if (!controller.signal.aborted) {
          setReport({
            ...result,
            trial_granted_cents: known?.granted ?? 0,
            trial_spent_cents: known?.spent ?? 0,
          });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setError(true);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [query, retry]);

  return (
    <div className="space-y-6 text-gray-800 dark:text-gray-100">
      <header className="space-y-2">
        <Heading level={1}>Statistics</Heading>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Track new organizations, trial credits, and plan activations.
        </p>
      </header>
      <RegistrationDateFilter period={query} onChange={(period) => setQuery({ ...period, offset: 0 })} />
      {loading ? (
        <p role="status" className="py-8 text-sm text-gray-500 dark:text-gray-400">
          Loading statistics...
        </p>
      ) : error ? (
        <div
          role="alert"
          className="space-y-3 rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900"
        >
          <p>Failed to load statistics. Try again.</p>
          <Button variant="outline" onClick={() => setRetry((value) => value + 1)}>
            Try again
          </Button>
        </div>
      ) : (
        report && (
          <RegistrationStatsReport
            report={report}
            periodStart={query.start}
            offset={query.offset}
            onPageChange={(offset) => setQuery((current) => ({ ...current, offset }))}
          />
        )
      )}
    </div>
  );
}

function RegistrationDateFilter({
  period,
  onChange,
}: {
  period: RegistrationPeriod;
  onChange: (period: RegistrationPeriod) => void;
}) {
  const [preset, setPreset] = useState<RegistrationPreset | "custom">("today");
  const [customStart, setCustomStart] = useState(() => dateInputValue(new Date()));
  const [customEnd, setCustomEnd] = useState(() => dateInputValue(new Date()));
  const [dateError, setDateError] = useState("");
  const changePreset = (value: RegistrationPreset | "custom") => {
    setPreset(value);
    setDateError("");
    if (value !== "custom") onChange(registrationPeriod(value));
  };

  function applyCustomDates() {
    const start = new Date(`${customStart}T00:00:00`);
    const end = new Date(`${customEnd}T00:00:00`);
    if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()) || start > end) {
      setDateError("Choose an end date on or after the start date.");
      return;
    }
    end.setDate(end.getDate() + 1);
    setDateError("");
    onChange({ start: start.toISOString(), end: end.toISOString() });
  }

  return (
    <section
      aria-label="Registration dates"
      className="space-y-3 rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900"
    >
      <div className="flex flex-wrap items-center gap-3">
        <Label htmlFor="registration-period">Period</Label>
        <Select value={preset} onValueChange={changePreset}>
          <SelectTrigger id="registration-period">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {presets.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {new Date(period.start).toLocaleDateString()} –{" "}
          {new Date(new Date(period.end).getTime() - 1).toLocaleDateString()}
        </p>
      </div>
      {preset === "custom" && (
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            applyCustomDates();
          }}
        >
          <div className="space-y-1">
            <Label htmlFor="stats-start">Start date</Label>
            <Input
              id="stats-start"
              type="date"
              required
              value={customStart}
              onChange={(event) => setCustomStart(event.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="stats-end">End date</Label>
            <Input
              id="stats-end"
              type="date"
              required
              min={customStart}
              value={customEnd}
              onChange={(event) => setCustomEnd(event.target.value)}
            />
          </div>
          <Button type="submit" variant="outline">
            Apply dates
          </Button>
          {dateError && (
            <p role="alert" className="text-sm text-red-600 dark:text-red-400">
              {dateError}
            </p>
          )}
        </form>
      )}
      <p className="text-xs text-gray-500 dark:text-gray-400">
        Registration totals use registration time. Plan activations use activation time. Dates use{" "}
        {Intl.DateTimeFormat().resolvedOptions().timeZone}. Weeks run Sunday to Saturday.
      </p>
    </section>
  );
}

function MetricCards({ metrics }: { metrics: { label: string; value: string }[] }) {
  return (
    <dl className="grid gap-4 sm:grid-cols-3">
      {metrics.map((metric) => (
        <div
          key={metric.label}
          className="rounded-lg border border-slate-200 bg-white p-5 dark:border-gray-700 dark:bg-gray-900"
        >
          <dt className="text-sm text-gray-500 dark:text-gray-400">{metric.label}</dt>
          <dd className="mt-2 text-2xl font-semibold tabular-nums">{metric.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function RegistrationStatsReport({
  report,
  periodStart,
  offset,
  onPageChange,
}: {
  report: RegistrationReport;
  periodStart: string;
  offset: number;
  onPageChange: (offset: number) => void;
}) {
  const registrationMetrics = [
    { label: "New organizations", value: report.total.toLocaleString() },
    { label: "Trial credits granted", value: formatUsdCents(report.trial_granted_cents) },
    { label: "Trial credits spent", value: formatUsdCents(report.trial_spent_cents) },
  ];
  const activationMetrics = [
    { label: "Team activations", value: report.plan_activations.team.toLocaleString() },
    { label: "Starter activations", value: report.plan_activations.starter.toLocaleString() },
    { label: "Pay-as-you-go activations", value: report.plan_activations.pay_as_you_go.toLocaleString() },
  ];
  const trackingStart = new Date(report.plan_activation_tracking_started_at);
  const trackingStartLabel = trackingStart.toLocaleDateString();
  const periodBeforeTracking = new Date(periodStart).getTime() < trackingStart.getTime();
  return (
    <>
      <MetricCards metrics={registrationMetrics} />
      <div className="space-y-2">
        <MetricCards metrics={activationMetrics} />
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Activations count each switch to a plan and each reactivation. Tracking started on {trackingStartLabel}.
        </p>
        {periodBeforeTracking && (
          <p role="note" className="text-sm text-amber-700 dark:text-amber-400">
            This period starts before tracking began. Activations before {trackingStartLabel} are not tracked, so these
            counts are incomplete.
          </p>
        )}
      </div>
      <section
        className="rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900"
        aria-labelledby="new-organizations-heading"
      >
        <h2 id="new-organizations-heading" className="text-base font-semibold">
          Newly registered organizations
        </h2>
        <p className="mt-1 mb-4 text-sm text-gray-500 dark:text-gray-400">
          Credit totals cover all organizations registered in this period, through today. Plans and available credits
          show current values.
        </p>
        <Table dense>
          <TableHead>
            <TableRow>
              <TableHeader>Name</TableHeader>
              <TableHeader>Owner email</TableHeader>
              <TableHeader>Plan</TableHeader>
              <TableHeader className="text-right">Available credits</TableHeader>
            </TableRow>
          </TableHead>
          <TableBody>
            {report.items.map((organization) => (
              <TableRow key={organization.id}>
                <TableCell className="max-w-64 whitespace-normal break-words">
                  <Link
                    className="font-medium text-blue-600 hover:underline dark:text-blue-400"
                    to={`/admin/organizations/${organization.id}`}
                  >
                    {organization.name}
                  </Link>
                </TableCell>
                <TableCell className="max-w-80 whitespace-normal break-words">
                  {organization.owner_email || "—"}
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{organization.plan}</Badge>
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatUsdCents(organization.available_credit_cents)}
                </TableCell>
              </TableRow>
            ))}
            {report.items.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="py-10 text-center whitespace-normal text-gray-500 dark:text-gray-400">
                  No organizations registered in this period. Choose a different date range.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
        <AdminPagination offset={offset} total={report.total} pageSize={PAGE_SIZE} onPageChange={onPageChange} />
      </section>
    </>
  );
}
