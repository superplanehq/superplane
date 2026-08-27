import type { FactoriesFactoryIntake } from "@/api-client";

import { lineIntakeSourceForApiSource } from "./lineIntakeModel";

export const BACKLOG_CREATE_COPY = {
  createWorkOrder: "Create work order",
  createManually: "Create task manually",
  empty: "No matching items.",
  loading: "Loading items.",
  loadingMore: "Loading more items.",
  unconnected: "Connect this intake first.",
} as const;

const INTAKE_NAME_SINGULAR: [RegExp, string][] = [
  [/ issues$/i, " issue"],
  [/ exceptions$/i, " exception"],
  [/ incidents$/i, " incident"],
];

export function searchPlaceholderForIntake(name: string): string {
  let singular = name.trim();
  for (const [pattern, replacement] of INTAKE_NAME_SINGULAR) {
    if (pattern.test(singular)) {
      singular = singular.replace(pattern, replacement);
      break;
    }
  }
  return `Import from ${singular}`;
}

export const LATEST_INTAKE_ITEMS = 5;
export const BACKLOG_SEARCH_PAGE_SIZE = LATEST_INTAKE_ITEMS;
export const BACKLOG_SEARCH_MAX_ITEMS = 50;

export interface BacklogIntakeItem {
  id: string;
  intakeId: string;
  key: string;
  title: string;
  body: string;
}

export interface BacklogIntakeItemCatalog {
  items: BacklogIntakeItem[];
  iconSrcByIntakeId?: Record<string, string>;
}

export interface BacklogIntakeSource {
  intakeId: string;
  name: string;
  iconSrc?: string;
  iconAlt: string;
}

export interface BacklogIntakeGroup extends BacklogIntakeSource {
  items: BacklogIntakeItem[];
}

export function listBacklogIntakeSources(args: {
  intakes: FactoriesFactoryIntake[];
  catalog?: Pick<BacklogIntakeItemCatalog, "iconSrcByIntakeId">;
}): BacklogIntakeSource[] {
  const sources: BacklogIntakeSource[] = [];

  for (const intake of args.intakes) {
    const source = sourceFromIntake(intake, args.catalog);
    if (source) {
      sources.push(source);
    }
  }

  return sources;
}

function sourceFromIntake(
  intake: FactoriesFactoryIntake,
  catalog?: Pick<BacklogIntakeItemCatalog, "iconSrcByIntakeId">,
): BacklogIntakeSource | undefined {
  const intakeId = intake.id?.trim();
  if (!intakeId) {
    return undefined;
  }

  const name = intake.name?.trim() || "Intake";
  const known = lineIntakeSourceForApiSource(intake.source);
  return {
    intakeId,
    name,
    iconSrc: catalog?.iconSrcByIntakeId?.[intakeId] ?? known?.iconSrc,
    iconAlt: known?.iconAlt ?? name,
  };
}

export function searchBacklogIntakeItems(args: {
  intakes: FactoriesFactoryIntake[];
  catalog: BacklogIntakeItemCatalog;
  query: string;
}): BacklogIntakeGroup[] {
  const needle = args.query.trim().toLowerCase();
  const groups: BacklogIntakeGroup[] = [];

  for (const intake of args.intakes) {
    const intakeId = intake.id?.trim();
    if (!intakeId) {
      continue;
    }

    const items = args.catalog.items.filter((item) => {
      if (item.intakeId !== intakeId) {
        return false;
      }
      if (!needle) {
        return true;
      }
      const haystack = `${item.key} ${item.title} ${item.body}`.toLowerCase();
      return haystack.includes(needle);
    });
    if (items.length === 0) {
      continue;
    }

    const source = sourceFromIntake(intake, args.catalog);
    if (!source) {
      continue;
    }
    groups.push({ ...source, items });
  }

  return groups;
}
