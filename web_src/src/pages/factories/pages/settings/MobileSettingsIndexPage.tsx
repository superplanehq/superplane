import { Input } from "@/components/ui/input";
import { useIsMobile } from "@/hooks/use-mobile";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { ArrowLeft, ChevronRight, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useParams } from "react-router";

import { factoryHomePath, factorySettingsSectionPath, firstFactoryLineId } from "../../lib/factoryPagePaths";
import { MOBILE_SETTINGS_COPY } from "../../mobile/mobileCopy";
import { factoryCardClassName } from "../factoryPageLayoutStyles";
import { LegacyFactorySettingsIndexRedirect } from "./FactorySettingsRedirects";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  factorySettingsNavGroupHeading,
  type FactorySettingsNavHeadingLabels,
  useFactorySettingsNavHeadingLabels,
} from "./settingsNavHeadings";
import type { FactorySettingsNavGroup } from "./settingsNavItems";
import {
  factorySettingsSearchResultPath,
  searchFactorySettings,
  type FactorySettingsSearchResult,
} from "./settingsSearch";
import { useFactorySettingsSearchIndex } from "./useFactorySettingsSearchIndex";
import { useVisibleFactorySettingsNavGroups } from "./useVisibleFactorySettingsNavGroups";

const ROW_CLASSNAME =
  "flex min-h-11 items-center gap-3 px-4 py-2.5 text-[15px] tracking-[-0.01em] text-foreground active:bg-sidebar-accent";

/**
 * Settings index route. Phones get a grouped list of every settings page,
 * the way native apps lay out Settings. Desktop keeps the redirect to the
 * first page, because the sidebar already lists every section.
 */
export function FactorySettingsIndexRoute() {
  if (useIsMobile()) {
    return <MobileSettingsIndexPage />;
  }
  return <LegacyFactorySettingsIndexRedirect />;
}

function MobileSettingsIndexPage() {
  const { organizationId, factory } = useFactorySettingsLayout();
  const { factoryKey: routeFactoryKey } = useParams<{ factoryKey: string }>();
  const factoryKey = routeFactoryKey ?? factory.key ?? "";
  const navGroups = useVisibleFactorySettingsNavGroups(organizationId);
  const searchIndex = useFactorySettingsSearchIndex(navGroups);
  const headingLabels = useFactorySettingsNavHeadingLabels(organizationId, factory, factoryKey);
  const [query, setQuery] = useState("");
  const searchResults = useMemo(() => searchFactorySettings(searchIndex, query), [query, searchIndex]);
  const isSearching = query.trim().length > 0;
  usePageTitle([MOBILE_SETTINGS_COPY.title, factory.name ?? "Workspace"]);

  return (
    <div className="flex min-h-full flex-col" data-testid="factory-settings-mobile-index">
      <header className="sticky top-0 z-10 shrink-0 border-b border-border bg-background pt-[env(safe-area-inset-top)]">
        <div className="flex h-12 items-center px-1">
          <Link
            to={factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory))}
            aria-label={MOBILE_SETTINGS_COPY.backToWorkspace}
            className="flex size-10 shrink-0 items-center justify-center rounded-md text-foreground hover:bg-accent"
            data-testid="factory-settings-mobile-index-back"
          >
            <ArrowLeft className="size-5" aria-hidden />
          </Link>
          <h1 className="min-w-0 flex-1 truncate text-center text-[15px] font-medium tracking-[-0.01em]">
            {MOBILE_SETTINGS_COPY.title}
          </h1>
          <div className="size-10 shrink-0" aria-hidden />
        </div>
        <div className="px-3 pb-3">
          <label className="sr-only" htmlFor="factory-settings-find">
            {MOBILE_SETTINGS_COPY.findPlaceholder}
          </label>
          <div className="relative">
            <Search
              className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden
            />
            <Input
              id="factory-settings-find"
              data-testid="factory-settings-find"
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={MOBILE_SETTINGS_COPY.findPlaceholder}
              className="!h-10 border-border bg-sidebar pl-9 text-[15px] shadow-none placeholder:text-muted-foreground dark:bg-sidebar"
            />
          </div>
        </div>
      </header>
      <div className="flex flex-col gap-6 px-3 pt-4 pb-10">
        {isSearching ? (
          <MobileSettingsSearchResults
            organizationId={organizationId}
            factoryKey={factoryKey}
            results={searchResults}
          />
        ) : (
          navGroups.map((group) => (
            <MobileSettingsGroup
              key={group.id}
              organizationId={organizationId}
              factoryKey={factoryKey}
              group={group}
              headingLabels={headingLabels}
            />
          ))
        )}
      </div>
    </div>
  );
}

function MobileSettingsGroup({
  organizationId,
  factoryKey,
  group,
  headingLabels,
}: {
  organizationId: string;
  factoryKey: string;
  group: FactorySettingsNavGroup;
  headingLabels: FactorySettingsNavHeadingLabels;
}) {
  const heading = factorySettingsNavGroupHeading(group.id, headingLabels);
  return (
    <section data-testid={`factory-settings-${group.id}-nav`}>
      <div className="mb-2 px-1" data-testid={heading.testId}>
        <h2 className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{heading.name}</h2>
        {heading.helper ? <p className="truncate text-[11px] text-muted-foreground">{heading.helper}</p> : null}
      </div>
      <ul className={cn(factoryCardClassName, "divide-y divide-border overflow-hidden")}>
        {group.items.map((item) => {
          const Icon = item.Icon;
          return (
            <li key={item.id}>
              <Link
                to={factorySettingsSectionPath(organizationId, factoryKey, item.scope, item.section)}
                className={ROW_CLASSNAME}
                data-testid={`factory-settings-nav-${item.id}`}
              >
                <Icon className="size-[18px] shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
                <span className="min-w-0 flex-1 truncate">{item.label}</span>
                <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function MobileSettingsSearchResults({
  organizationId,
  factoryKey,
  results,
}: {
  organizationId: string;
  factoryKey: string;
  results: FactorySettingsSearchResult[];
}) {
  if (results.length === 0) {
    return (
      <p className="px-1 text-[13px] text-muted-foreground" data-testid="factory-settings-find-empty">
        {MOBILE_SETTINGS_COPY.noResults}
      </p>
    );
  }

  return (
    <ul
      className={cn(factoryCardClassName, "divide-y divide-border overflow-hidden")}
      data-testid="factory-settings-search-results"
    >
      {results.map((result) => (
        <li key={result.id}>
          <Link
            to={factorySettingsSearchResultPath(organizationId, factoryKey, result, factorySettingsSectionPath)}
            className={ROW_CLASSNAME}
            data-testid={`factory-settings-search-${result.id}`}
          >
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="truncate">{result.title}</span>
              <span className="truncate text-[12px] text-muted-foreground">{result.breadcrumb.join(" › ")}</span>
            </span>
            <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          </Link>
        </li>
      ))}
    </ul>
  );
}
