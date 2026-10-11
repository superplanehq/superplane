import { Avatar } from "@/components/Avatar/avatar";
import { FeedbackDialog } from "@/components/FeedbackDialog";
import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAccount } from "@/contexts/useAccount";
import { useTheme } from "@/contexts/useTheme";
import { useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useFactoryAutomations } from "@/hooks/useFactoryData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { isThemePreference } from "@/lib/themePreference";
import type { FeedbackCategory } from "@/lib/submitFeedback";
import { posthog } from "@/posthog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowRightLeft, Bug, ChevronRight, CreditCard, LogOut, MessageSquare, Settings, Shield } from "lucide-react";
import type { FactoriesFactoryLine } from "@/api-client";
import { useId, useMemo, useState, type ReactNode } from "react";
import { Link, useLocation } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { initialsForName } from "../layout/factoriesRail";
import {
  factorySettingsPath,
  factorySettingsSectionPath,
  firstFactoryLineId,
  workOrderBoardLineIdFromSearch,
} from "../lib/factoryPagePaths";
import {
  isLineBoardColumnColorView,
  useLineBoardColumnColorViewPreference,
} from "../lib/lineBoardColumnColorViewPreference";
import { ColumnAutomationGlyph } from "../pages/ColumnAutomationsPopup";
import { useHostedCreditChrome } from "../lib/useHostedCreditEmptyBanner";
import {
  buildColumnAutomations,
  COLUMN_AUTOMATIONS_COPY,
  columnAutomationOpenPath,
  columnTitleForKey,
  type ColumnAutomation,
  type ColumnKey,
} from "../lib/columnAutomations";
import { columnAutomationHeadline } from "../lib/columnAutomationHeadline";
import { LINE_INTAKE_COPY } from "../pages/lineIntakeModel";
import { buildLinePhaseBoard } from "../lib/linePhaseRuns";
import { MobileWorkspaceSwitcher } from "./MobileWorkspaceSwitcher";

const ROW_CLASS =
  "flex min-h-14 w-full items-center gap-3 px-4 py-3 text-left text-sm font-medium hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&>svg]:size-5 [&>svg]:shrink-0 [&>svg]:text-muted-foreground";
const ROW_BUTTON_CLASS =
  "h-auto min-h-14 w-full justify-start gap-3 rounded-none px-4 py-3 text-left whitespace-normal hover:bg-accent";
const GROUP_CLASS = "overflow-hidden rounded-lg border border-border bg-card divide-y divide-border";

export function MobileMorePage() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  const { preference, setPreference } = useTheme();
  const appearanceId = useId();
  const { headerKicker } = useHostedCreditChrome(organizationId, routeSegment);
  const [feedbackCategory, setFeedbackCategory] = useState<FeedbackCategory>();
  usePageTitle(["More", factory?.name ?? "Workspace"]);
  const settingsHref = (scope: "workspace" | "organization" | "account", section: string) =>
    factorySettingsSectionPath(organizationId, routeSegment, scope, section);

  return (
    <div className="space-y-5 px-4 pt-[env(safe-area-inset-top)] pb-6" data-testid="mobile-more-page">
      <div className="flex h-16 items-center">
        <MobileWorkspaceSwitcher />
      </div>
      <MobileAccountCard
        href={settingsHref("account", "general")}
        name={account?.name}
        avatarUrl={account?.avatar_url}
      />
      {headerKicker ? <div className="flex flex-wrap items-center gap-2">{headerKicker}</div> : null}
      <div className={GROUP_CLASS}>
        <MoreLink href={factorySettingsPath(organizationId, routeSegment)} icon={<Settings />}>
          Settings
        </MoreLink>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="ghost" className={ROW_BUTTON_CLASS}>
              <ArrowRightLeft className="size-5 text-muted-foreground" aria-hidden />
              <span className="min-w-0 flex-1 truncate">{organization?.metadata?.name || "Organization"}</span>
              <span className="text-xs font-normal text-muted-foreground">Switch</span>
              <ChevronRight className="size-5 text-muted-foreground" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-[min(22rem,calc(100vw-2rem))] [&_[role=menuitem]]:min-h-11">
            <OrganizationSwitchMenu currentOrganizationRouteId={organizationId} />
          </DropdownMenuContent>
        </DropdownMenu>
        {account?.installation_admin ? (
          <MoreLink href="/admin" icon={<Shield />}>
            Installation Admin
          </MoreLink>
        ) : null}
        <MoreLink href={settingsHref("organization", "billing")} icon={<CreditCard />}>
          Billing
        </MoreLink>
        <div className="flex min-h-14 items-center justify-between gap-3 px-4 py-2">
          <Label htmlFor={appearanceId}>Appearance</Label>
          <Select
            value={preference}
            onValueChange={(value) => {
              if (isThemePreference(value)) setPreference(value);
            }}
          >
            <SelectTrigger id={appearanceId} className="!h-11 w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="[&_[role=option]]:min-h-11">
              {["light", "dark", "system"].map((value) => (
                <SelectItem key={value} value={value}>
                  {value.charAt(0).toUpperCase() + value.slice(1)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <MobileBoardOptions />
      <div className={GROUP_CLASS}>
        <Button type="button" variant="ghost" className={ROW_BUTTON_CLASS} onClick={() => setFeedbackCategory("bug")}>
          <Bug className="size-5 text-muted-foreground" aria-hidden />
          Report issue
        </Button>
        <Button type="button" variant="ghost" className={ROW_BUTTON_CLASS} onClick={() => setFeedbackCategory("other")}>
          <MessageSquare className="size-5 text-muted-foreground" aria-hidden />
          Send feedback
        </Button>
      </div>
      <Button
        type="button"
        variant="ghost"
        className={`${ROW_BUTTON_CLASS} rounded-lg border border-border bg-card`}
        onClick={() => {
          posthog.reset();
          window.location.href = "/logout";
        }}
      >
        <LogOut className="size-5 text-muted-foreground" aria-hidden />
        Sign out
      </Button>
      <FeedbackDialog
        open={feedbackCategory !== undefined}
        onOpenChange={(open) => {
          if (!open) setFeedbackCategory(undefined);
        }}
        organizationId={organizationId}
        initialCategory={feedbackCategory}
      />
    </div>
  );
}

function MobileAccountCard({
  href,
  name,
  avatarUrl,
}: {
  href: string;
  name?: string | null;
  avatarUrl?: string | null;
}) {
  const displayName = name || "Your account";
  return (
    <Link to={href} className="flex items-center gap-3 rounded-lg border border-border bg-card p-4 hover:bg-accent">
      <Avatar
        src={avatarUrl || undefined}
        initials={avatarUrl ? undefined : initialsForName(displayName)}
        alt={displayName}
        className="size-10"
      />
      <span className="min-w-0">
        <p className="truncate text-base font-semibold">{displayName}</p>
        <p className="mt-1 text-sm text-muted-foreground">Profile and account settings</p>
      </span>
    </Link>
  );
}

function MoreLink({ href, icon, children }: { href: string; icon: ReactNode; children: ReactNode }) {
  return (
    <Link to={href} className={ROW_CLASS}>
      {icon}
      <span className="min-w-0 flex-1">{children}</span>
      <ChevronRight aria-hidden />
    </Link>
  );
}

function MobileBoardOptions() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const { search } = useLocation();
  const { line, lineId } = useMobileBoardOptionLine(factory?.lines, workOrderBoardLineIdFromSearch(search));
  const intakes = useFactoryIntakes(organizationId, factoryId);
  const handlers = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const automations = useFactoryAutomations(organizationId, factoryId);
  const { view, setView } = useLineBoardColumnColorViewPreference();
  const columnColorId = useId();
  const columns = useMobileBoardOptionColumns({
    line,
    apps: automations.data,
    intakes: intakes.data,
    handlers: handlers.data,
    planningEnabled: factory?.planning?.enabled !== false,
    vcsProvider: factory?.onboarding?.vcsProvider,
  });
  const retry = () => {
    void intakes.refetch();
    void handlers.refetch();
    void automations.refetch();
  };
  return (
    <section aria-label="Board options" className="space-y-3">
      <h2 className="text-sm font-medium text-muted-foreground">Board options</h2>
      <div className={GROUP_CLASS}>
        <div className="flex min-h-14 items-center justify-between gap-3 px-4 py-2">
          <Label htmlFor={columnColorId}>Column colors</Label>
          <Select
            value={view}
            onValueChange={(value) => {
              if (isLineBoardColumnColorView(value)) setView(value);
            }}
          >
            <SelectTrigger id={columnColorId} className="!h-11 w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="[&_[role=option]]:min-h-11">
              <SelectItem value="vivid">Vivid</SelectItem>
              <SelectItem value="dim">Soft</SelectItem>
              <SelectItem value="borders">Borders</SelectItem>
              <SelectItem value="off">Off</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <MobileBoardOptionsStatus
          isLoading={intakes.isLoading || handlers.isLoading || automations.isLoading}
          isError={intakes.isError || handlers.isError || automations.isError}
          onRetry={retry}
        />
      </div>
      <MobileBoardOptionColumnSections
        columns={columns}
        organizationId={organizationId}
        factoryKey={routeSegment}
        lineId={lineId}
      />
    </section>
  );
}

function MobileBoardOptionsStatus({
  isLoading,
  isError,
  onRetry,
}: {
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}) {
  if (isLoading) {
    return (
      <p role="status" className="p-4 text-sm text-muted-foreground">
        Loading board options…
      </p>
    );
  }
  if (isError) {
    return (
      <div role="alert" className="p-4 text-sm">
        <p>SuperPlane could not load board options.</p>
        <Button variant="outline" className="mt-2 h-11" onClick={onRetry}>
          Try again
        </Button>
      </div>
    );
  }
  return null;
}

function MobileBoardOptionColumnSections({
  columns,
  organizationId,
  factoryKey,
  lineId,
}: {
  columns: Array<{ key: ColumnKey; title: string; automations: ColumnAutomation[] }>;
  organizationId: string;
  factoryKey: string;
  lineId: string | undefined;
}) {
  return (
    <>
      {columns.map((column) => (
        <div key={column.key} className="space-y-3">
          <h3 className="text-sm font-medium text-muted-foreground">{column.title}</h3>
          <div className={GROUP_CLASS}>
            {column.automations.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">No automations in {column.title}.</p>
            ) : (
              column.automations.map((automation) => (
                <MobileAutomationRow
                  key={automation.id}
                  automation={automation}
                  href={columnAutomationOpenPath(automation, {
                    organizationId,
                    factoryKey,
                    lineId,
                  })}
                />
              ))
            )}
          </div>
        </div>
      ))}
    </>
  );
}

function useMobileBoardOptionLine(lines: FactoriesFactoryLine[] | undefined, requestedLineId: string | null) {
  return useMemo(() => {
    const list = lines ?? [];
    const line = list.find((l) => l.id === requestedLineId) ?? list[0];
    return {
      line,
      lineId: line?.id ?? firstFactoryLineId({ lines: list }),
    };
  }, [lines, requestedLineId]);
}

function useMobileBoardOptionColumns({
  line,
  apps,
  intakes,
  handlers,
  planningEnabled,
  vcsProvider,
}: {
  line?: FactoriesFactoryLine;
  apps?: Array<{ id?: string; name?: string; columnKey?: string }>;
  intakes?: Parameters<typeof buildColumnAutomations>[1]["intakes"];
  handlers?: Parameters<typeof buildColumnAutomations>[1]["prFeedbackHandlers"];
  planningEnabled: boolean;
  vcsProvider?: string;
}) {
  const normalizedApps = useMemo(() => apps ?? [], [apps]);
  const board = useMemo(() => (line ? buildLinePhaseBoard(line, [], normalizedApps) : []), [line, normalizedApps]);
  return useMemo(() => {
    const keys: ColumnKey[] = [
      "backlog",
      ...board.map((column): ColumnKey => `phase-${column.stepIndex}`),
      "verify",
      "done",
    ];
    return keys.map((key) => ({
      key,
      title: columnTitleForKey(key, board),
      automations: mobileColumnAutomations(key, {
        columnTitle: columnTitleForKey(key, board),
        columns: board,
        intakes: intakes ?? [],
        prFeedbackHandlers: handlers ?? [],
        apps: normalizedApps,
        workOrders: [],
        planningEnabled,
        vcsProvider,
      }),
    }));
  }, [board, handlers, intakes, normalizedApps, planningEnabled, vcsProvider]);
}

type MobileColumnAutomation = ColumnAutomation & { paused?: boolean };

function mobileColumnAutomations(
  key: ColumnKey,
  input: Parameters<typeof buildColumnAutomations>[1],
): MobileColumnAutomation[] {
  const intakes = input.intakes ?? [];
  return buildColumnAutomations(key, input).map((automation) => {
    if (automation.kind !== "intake") {
      return automation;
    }
    const intake = intakes.find((item) => item.id === automation.id);
    if (!intake) {
      return automation;
    }
    return {
      ...automation,
      health: intake.healthy === false ? "needs-repair" : automation.health,
      paused: intake.paused === true,
    };
  });
}

function MobileAutomationRow({ automation, href }: { automation: MobileColumnAutomation; href: string | undefined }) {
  const icon = <ColumnAutomationGlyph automation={automation} className="size-5" />;
  const label = (
    <span>
      <span className="block">{automation.name}</span>
      <span className="block text-xs font-normal text-muted-foreground">{columnAutomationHeadline(automation)}</span>
      {automation.health === "needs-repair" ? (
        <span className="block text-xs text-amber-700 dark:text-amber-400">
          {COLUMN_AUTOMATIONS_COPY.needsRepairLabel}
        </span>
      ) : null}
      {automation.paused ? (
        <span className="block text-xs text-muted-foreground" title={LINE_INTAKE_COPY.pausedHelper}>
          {LINE_INTAKE_COPY.paused}
        </span>
      ) : null}
      {automation.health === "disabled" ? (
        <span className="block text-xs text-muted-foreground">{COLUMN_AUTOMATIONS_COPY.disabledLabel}</span>
      ) : null}
    </span>
  );
  if (!href) {
    return (
      <div className={ROW_CLASS}>
        {icon}
        <span className="min-w-0 flex-1">{label}</span>
      </div>
    );
  }
  return (
    <MoreLink href={href} icon={icon}>
      {label}
    </MoreLink>
  );
}
