import { ChevronsUpDown, Globe, X } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAdminOrganizationSearch } from "@/hooks/useAdminIntakeCatalog";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

import { formatDate } from "../formatDate";
import {
  INTERNAL_ALL_COMPANIES_REASON,
  NO_COMPANIES_EMPTY,
  NOT_IMPLEMENTED_REASON,
  ONLY_BETA_FOR_ALL_REASON,
  type AdminIntakeEntry,
} from "./intakeCatalogModel";

interface IntakeAccessCardProps {
  entry: AdminIntakeEntry;
  pending: boolean;
  onSetEnabledForAll: (enabled: boolean) => void;
  onAddOrganization: (organizationId: string) => void;
  onRemoveOrganization: (organizationId: string) => void;
}

export function IntakeAccessCard(props: IntakeAccessCardProps) {
  const { entry } = props;

  if (!entry.implemented) {
    return <AccessMessage>{NOT_IMPLEMENTED_REASON} Nobody can use it.</AccessMessage>;
  }
  if (entry.status === "ga") {
    return (
      <AccessMessage>
        <Globe size={14} className="shrink-0 text-emerald-600" />
        All companies can use this intake.
      </AccessMessage>
    );
  }

  return <AccessControls {...props} />;
}

function AccessControls({
  entry,
  pending,
  onSetEnabledForAll,
  onAddOrganization,
  onRemoveOrganization,
}: IntakeAccessCardProps) {
  const allCompanies = entry.enabled_for_all && entry.status === "beta";
  const statusHint = accessStatusHint(entry.status);

  return (
    <div className="flex flex-col gap-3">
      <AccessModeControl
        allCompanies={allCompanies}
        allowAll={entry.status === "beta"}
        allDisabledReason={entry.status === "alpha" ? INTERNAL_ALL_COMPANIES_REASON : ONLY_BETA_FOR_ALL_REASON}
        disabled={pending}
        onChange={onSetEnabledForAll}
      />
      {statusHint ? <p className="text-xs text-slate-500 dark:text-gray-400">{statusHint}</p> : null}

      {allCompanies ? (
        <p className="text-sm text-slate-600 dark:text-gray-300">
          This is an open beta. All companies can use this intake.
        </p>
      ) : (
        <>
          <CompanyPicker
            excluded={entry.organizations.map((organization) => organization.id)}
            disabled={pending}
            onPick={onAddOrganization}
          />
          <CompanyList entry={entry} disabled={pending} onRemove={onRemoveOrganization} />
        </>
      )}
    </div>
  );
}

function accessStatusHint(status: string): string {
  if (status === "planned") {
    return "Nobody can use a Planned intake. The companies that you add get access when the status is Internal or Beta.";
  }
  if (status === "deprecated") {
    return "Nobody can create a new intake of a Deprecated type. The company list applies again after you restore it.";
  }
  return "";
}

interface AccessModeControlProps {
  allCompanies: boolean;
  allowAll: boolean;
  allDisabledReason: string;
  disabled: boolean;
  onChange: (enabled: boolean) => void;
}

function AccessModeControl({ allCompanies, allowAll, allDisabledReason, disabled, onChange }: AccessModeControlProps) {
  const segment = (active: boolean) =>
    cn(
      "flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
      active
        ? "bg-white text-slate-900 shadow-xs dark:bg-gray-700 dark:text-gray-100"
        : "text-slate-500 hover:text-slate-800 dark:text-gray-400 dark:hover:text-gray-100",
    );

  const allButton = (
    <button
      type="button"
      role="radio"
      aria-checked={allCompanies}
      disabled={disabled || !allowAll}
      onClick={() => onChange(true)}
      className={cn(segment(allCompanies), !allowAll && "cursor-not-allowed opacity-50")}
    >
      All companies
    </button>
  );

  return (
    <div role="radiogroup" aria-label="Access" className="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-gray-800">
      <button
        type="button"
        role="radio"
        aria-checked={!allCompanies}
        disabled={disabled}
        onClick={() => onChange(false)}
        className={segment(!allCompanies)}
      >
        Selected companies
      </button>
      {allowAll ? (
        allButton
      ) : (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex flex-1" tabIndex={0}>
              {allButton}
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-64">{allDisabledReason}</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

interface CompanyPickerProps {
  excluded: string[];
  disabled: boolean;
  onPick: (organizationId: string) => void;
}

function CompanyPicker({ excluded, disabled, onPick }: CompanyPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const { data: organizations = [], isFetching } = useAdminOrganizationSearch(search, open);
  const options = organizations.filter((organization) => !excluded.includes(organization.id));

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" disabled={disabled} className="w-full justify-between rounded-md font-normal">
          <span className="text-slate-500 dark:text-gray-400">Search companies to add</span>
          <ChevronsUpDown size={14} className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--radix-popover-trigger-width) p-0">
        <Command shouldFilter={false}>
          <CommandInput placeholder="Company name" value={search} onValueChange={setSearch} />
          <CommandList>
            <CommandEmpty>{isFetching ? "Searching..." : "No companies found."}</CommandEmpty>
            <CommandGroup>
              {options.map((organization) => (
                <CommandItem
                  key={organization.id}
                  value={organization.id}
                  onSelect={() => {
                    onPick(organization.id);
                    setOpen(false);
                    setSearch("");
                  }}
                >
                  {organization.name}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

interface CompanyListProps {
  entry: AdminIntakeEntry;
  disabled: boolean;
  onRemove: (organizationId: string) => void;
}

function CompanyList({ entry, disabled, onRemove }: CompanyListProps) {
  if (entry.organizations.length === 0) {
    return (
      <p className="rounded-md border border-dashed border-slate-300 px-3 py-4 text-center text-sm text-slate-500 dark:border-gray-700 dark:text-gray-400">
        {NO_COMPANIES_EMPTY}
      </p>
    );
  }

  return (
    <ul
      aria-label="Companies with access"
      className="divide-y divide-slate-100 rounded-md border border-slate-200 dark:divide-gray-800 dark:border-gray-700"
    >
      {entry.organizations.map((organization) => (
        <li key={organization.id} className="flex items-center gap-3 px-3 py-2 text-sm">
          <span className="flex-1 truncate font-medium text-slate-800 dark:text-gray-100">{organization.name}</span>
          <span className="text-xs text-slate-400 dark:text-gray-500">Added {formatDate(organization.added_at)}</span>
          <Button
            variant="ghost"
            size="icon-xs"
            disabled={disabled}
            aria-label={`Remove ${organization.name}`}
            onClick={() => onRemove(organization.id)}
          >
            <X size={14} />
          </Button>
        </li>
      ))}
    </ul>
  );
}

function AccessMessage({ children }: { children: React.ReactNode }) {
  return (
    <p className="flex items-center gap-2 rounded-md bg-slate-50 px-3 py-2.5 text-sm text-slate-600 dark:bg-gray-800/60 dark:text-gray-300">
      {children}
    </p>
  );
}
