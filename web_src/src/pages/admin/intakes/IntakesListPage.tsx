import { ChevronRight, Plus, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAdminIntakeCatalog } from "@/hooks/useAdminIntakeCatalog";
import { usePageTitle } from "@/hooks/usePageTitle";
import { INTAKE_CATEGORIES, INTAKE_STATUSES, type IntakeCategory, type IntakeStatus } from "@/lib/intakeCatalog";

import { CreateIntakeDialog } from "./CreateIntakeDialog";
import { countBy, filterIntakeEntries, intakeCategoryLabel, type AdminIntakeEntry } from "./intakeCatalogModel";
import { CategoryRail, StatusGuide, StatusRail, SurfaceIcons } from "./IntakeListParts";
import { IntakeIcon, IntakeStatusPill } from "./IntakeStatusPill";

export function IntakesListPage() {
  usePageTitle(["Intakes", "Installation Admin"]);
  const navigate = useNavigate();
  const { data: entries = [], isLoading, isError } = useAdminIntakeCatalog();
  const [status, setStatus] = useState<IntakeStatus | null>(null);
  const [category, setCategory] = useState<IntakeCategory | null>(null);
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  const categoryCounts = useMemo(() => countBy(entries, INTAKE_CATEGORIES, (entry) => entry.category), [entries]);
  const statusCounts = useMemo(() => countBy(entries, INTAKE_STATUSES, (entry) => entry.status), [entries]);
  const visible = useMemo(
    () => filterIntakeEntries(entries, { status, category, search }),
    [entries, status, category, search],
  );
  const clearFilters = () => {
    setStatus(null);
    setCategory(null);
    setSearch("");
  };

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-gray-100">Intakes</h1>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <p className="text-sm text-slate-500 dark:text-gray-400">Control the maturity label for each intake.</p>
            <StatusGuide />
          </div>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search size={14} className="absolute top-1/2 left-2.5 -translate-y-1/2 text-slate-400" />
            <Input
              value={search}
              onChange={(event: React.ChangeEvent<HTMLInputElement>) => setSearch(event.target.value)}
              placeholder="Search intakes"
              aria-label="Search intakes"
              className="h-9 w-56 bg-white pl-8 dark:bg-gray-900"
            />
          </div>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus size={14} />
            Add intake
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-[13rem_1fr] gap-6">
        <div className="flex flex-col gap-6">
          <StatusRail counts={statusCounts} total={entries.length} selected={status} onSelect={setStatus} />
          <CategoryRail counts={categoryCounts} total={entries.length} selected={category} onSelect={setCategory} />
        </div>
        <IntakeGroups entries={visible} loading={isLoading} failed={isError} onClearFilter={clearFilters} />
      </div>

      <CreateIntakeDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(entry) => navigate(`/admin/intakes/${encodeURIComponent(entry.key)}`)}
      />
    </div>
  );
}

interface IntakeGroupsProps {
  entries: AdminIntakeEntry[];
  loading: boolean;
  failed: boolean;
  onClearFilter: () => void;
}

function IntakeGroups({ entries, loading, failed, onClearFilter }: IntakeGroupsProps) {
  if (loading) {
    return <ListMessage>Loading intakes...</ListMessage>;
  }
  if (failed) {
    return <ListMessage>Cannot load the intakes. Refresh the page to try again.</ListMessage>;
  }
  if (entries.length === 0) {
    return (
      <ListMessage>
        No intakes match this filter.{" "}
        <button
          type="button"
          onClick={onClearFilter}
          className="font-medium text-slate-900 underline dark:text-gray-100"
        >
          Clear filter
        </button>
      </ListMessage>
    );
  }

  const groups = INTAKE_CATEGORIES.map((category) => ({
    category,
    entries: entries.filter((entry) => entry.category === category),
  })).filter((group) => group.entries.length > 0);

  return (
    <div className="flex flex-col gap-5">
      {groups.map((group) => (
        <section key={group.category} aria-label={intakeCategoryLabel(group.category)}>
          <h2 className="pb-2 text-[11px] font-semibold tracking-wide text-slate-400 uppercase dark:text-gray-500">
            {intakeCategoryLabel(group.category)}
          </h2>
          <ul className="divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white dark:divide-gray-800 dark:border-gray-700 dark:bg-gray-900">
            {group.entries.map((entry) => (
              <IntakeRow key={entry.key} entry={entry} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

function IntakeRow({ entry }: { entry: AdminIntakeEntry }) {
  const note = entry.status_note.split("\n")[0];
  return (
    <li>
      <Link
        to={`/admin/intakes/${encodeURIComponent(entry.key)}`}
        className="group flex items-center gap-3 px-4 py-3 transition-colors hover:bg-slate-50 dark:hover:bg-gray-800/60"
      >
        <IntakeIcon intakeKey={entry.key} name={entry.name} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{entry.name}</span>
            <IntakeStatusPill status={entry.status} />
          </div>
          {note ? <p className="truncate text-xs text-slate-500 dark:text-gray-400">{note}</p> : null}
        </div>
        <SurfaceIcons intakeKey={entry.key} category={entry.category} />
        <ChevronRight size={16} className="text-slate-300 group-hover:text-slate-500 dark:text-gray-600" />
      </Link>
    </li>
  );
}

function ListMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center text-sm text-slate-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-400">
      {children}
    </div>
  );
}
