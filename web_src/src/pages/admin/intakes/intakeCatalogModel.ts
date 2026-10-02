import { INTAKE_CATEGORIES, INTAKE_STATUSES, type IntakeCategory, type IntakeStatus } from "@/lib/intakeCatalog";
import type { IntakeSurface } from "@/lib/intakePresentation";

export interface IntakeStatusInfo {
  label: string;
  summary: string;
  whoCanUse: string;
  /** The condition to move to the next step. Empty for the last steps. */
  nextStep: string;
  pillClassName: string;
  dotClassName: string;
}

export const INTAKE_STATUS_INFO: Record<IntakeStatus, IntakeStatusInfo> = {
  planned: {
    label: "Planned",
    summary: "The intake is not built yet. We plan to build it.",
    whoCanUse: "Nobody. Companies see Coming soon.",
    nextStep: "The code is merged and works for SuperPlane.",
    pillClassName:
      "border-slate-300 bg-slate-100 text-slate-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200",
    dotClassName: "bg-slate-400",
  },
  alpha: {
    label: "Internal",
    summary: "SuperPlane uses the intake every day to test it.",
    whoCanUse: "Only the companies that you add. Add SuperPlane companies only.",
    nextStep: "It works for our own work and the known errors are fixed.",
    pillClassName:
      "border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-800 dark:bg-violet-950 dark:text-violet-200",
    dotClassName: "bg-violet-500",
  },
  beta: {
    label: "Beta",
    summary: "Outside testers use the intake and send feedback. We do not promise that it works.",
    whoCanUse: "The companies that you add, or all companies for an open beta.",
    nextStep: "Outside testers use it with no new errors, and we are ready to support paying customers.",
    pillClassName:
      "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200",
    dotClassName: "bg-amber-500",
  },
  ga: {
    label: "Generally available",
    summary: "The intake is ready for paying customers. We support it.",
    whoCanUse: "All companies.",
    nextStep: "",
    pillClassName:
      "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-200",
    dotClassName: "bg-emerald-500",
  },
  deprecated: {
    label: "Deprecated",
    summary: "We plan to remove the intake. Existing intakes continue to run.",
    whoCanUse: "Nobody can create a new intake. Existing intakes continue to run.",
    nextStep: "",
    pillClassName: "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-200",
    dotClassName: "bg-red-500",
  },
};

/** The steps of the maturity stepper. Deprecated is a separate action. */
export const MATURITY_STEPS: readonly IntakeStatus[] = ["planned", "alpha", "beta", "ga"];

export const INTAKE_CATEGORY_LABELS: Record<IntakeCategory, string> = {
  issue_tracking: "Issue tracking",
  error_tracking: "Error tracking",
  incident_management: "Incident management",
  security_alerts: "Security alerts",
  repository_provider: "Repository providers",
};

export const INTAKE_SURFACE_LABELS: Record<IntakeSurface, string> = {
  addIntake: "Add intake",
  onboardingTickets: "Onboarding: tickets",
  onboardingRepository: "Onboarding: repository",
};

export const CREATE_INTAKE_HELP = "New intakes start as Planned. Nobody can use a Planned intake.";
export const STATUS_NOTE_HELP = "Tell other admins what works, what is tested, and what is known to fail.";
export const INTERNAL_ALL_COMPANIES_REASON =
  "Internal intakes are only for SuperPlane companies. Move the intake to Beta to give access to all companies.";
export const ONLY_BETA_FOR_ALL_REASON = "Only a Beta intake can be open to all companies.";
export const NOT_IMPLEMENTED_REASON = "The code for this intake does not exist yet. The status stays Planned.";
export const NO_COMPANIES_EMPTY = "No companies can use this intake. Add a company to start.";

export function isIntakeStatus(value: string): value is IntakeStatus {
  return (INTAKE_STATUSES as readonly string[]).includes(value);
}

export function isIntakeCategory(value: string): value is IntakeCategory {
  return (INTAKE_CATEGORIES as readonly string[]).includes(value);
}

export function intakeStatusInfo(status: string): IntakeStatusInfo {
  return isIntakeStatus(status) ? INTAKE_STATUS_INFO[status] : INTAKE_STATUS_INFO.planned;
}

export function intakeCategoryLabel(category: string): string {
  return isIntakeCategory(category) ? INTAKE_CATEGORY_LABELS[category] : category;
}

export interface AdminIntakeOrganization {
  id: string;
  name: string;
  added_at: string;
}

export interface AdminIntakeEntry {
  key: string;
  name: string;
  category: string;
  status: string;
  status_note: string;
  enabled_for_all: boolean;
  implemented: boolean;
  deletable: boolean;
  created_at: string;
  updated_at: string;
  updated_by_name: string;
  organizations: AdminIntakeOrganization[];
}

/** One line that tells which companies can use the entry. */
export function intakeAccessSummary(entry: AdminIntakeEntry): string {
  if (!entry.implemented) {
    return "Not implemented";
  }
  if (entry.status === "planned" || entry.status === "deprecated") {
    return "No new intakes";
  }
  if (entry.status === "ga" || entry.enabled_for_all) {
    return "All companies";
  }
  const count = entry.organizations.length;
  if (count === 0) {
    return "No companies";
  }
  return count === 1 ? "1 company" : `${count} companies`;
}

/**
 * Tells if a move to the next status gives access to more companies. The
 * admin must confirm such a change.
 */
export function widensAccess(from: string, to: string): boolean {
  const rank: Record<IntakeStatus, number> = { deprecated: 0, planned: 0, alpha: 1, beta: 2, ga: 3 };
  if (!isIntakeStatus(from) || !isIntakeStatus(to)) {
    return false;
  }
  return rank[to] > rank[from];
}

/** A key from a name: lowercase, with dashes between words. */
export function intakeKeyFromName(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}

export interface IntakeListFilter {
  status: IntakeStatus | null;
  category: IntakeCategory | null;
  search: string;
}

export function filterIntakeEntries(
  entries: readonly AdminIntakeEntry[],
  filter: IntakeListFilter,
): AdminIntakeEntry[] {
  const search = filter.search.trim().toLowerCase();
  return entries.filter((entry) => {
    if (filter.status && entry.status !== filter.status) return false;
    if (filter.category && entry.category !== filter.category) return false;
    if (!search) return true;
    return entry.name.toLowerCase().includes(search) || entry.key.includes(search);
  });
}

export function countBy<T extends string>(
  entries: readonly AdminIntakeEntry[],
  values: readonly T[],
  pick: (entry: AdminIntakeEntry) => string,
): Record<T, number> {
  const counts = Object.fromEntries(values.map((value) => [value, 0])) as Record<T, number>;
  for (const entry of entries) {
    const value = pick(entry);
    if ((values as readonly string[]).includes(value)) {
      counts[value as T] += 1;
    }
  }
  return counts;
}
