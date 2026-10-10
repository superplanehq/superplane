import githubIcon from "@/assets/icons/integrations/github.svg";
import { FeedbackDialog } from "@/components/FeedbackDialog";
import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAccount } from "@/contexts/useAccount";
import { useTheme } from "@/contexts/useTheme";
import { useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { isThemePreference } from "@/lib/themePreference";
import type { FeedbackCategory } from "@/lib/submitFeedback";
import { posthog } from "@/posthog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowRightLeft, Bug, ChevronRight, CreditCard, LogOut, MessageSquare, Settings, Shield } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import { Link, useLocation } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  factoryIntakePath,
  factoryPRFeedbackPath,
  factorySettingsPath,
  factorySettingsSectionPath,
  firstFactoryLineId,
  workOrderBoardLineIdFromSearch,
} from "../lib/factoryPagePaths";
import {
  isLineBoardColumnColorView,
  useLineBoardColumnColorViewPreference,
} from "../lib/lineBoardColumnColorViewPreference";
import { useHostedCreditChrome } from "../lib/useHostedCreditEmptyBanner";
import { intakeSourcesFromFactoryIntakes, lineIntakeListenTitle } from "../pages/lineIntakeModel";
import { prFeedbackListenTitle } from "../pages/prFeedbackSettingsModel";
import { ColumnAutomationGlyph } from "../pages/ColumnAutomationsPopup";
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
      <h1 className="text-2xl font-semibold tracking-tight">More</h1>
      <Link
        to={settingsHref("account", "general")}
        className="block rounded-lg border border-border bg-card p-4 hover:bg-accent"
      >
        <p className="text-base font-semibold">{account?.name || "Your account"}</p>
        <p className="mt-1 text-sm text-muted-foreground">Profile and account settings</p>
      </Link>
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
  const requestedLineId = workOrderBoardLineIdFromSearch(search);
  const lineId = factory?.lines?.find((line) => line.id === requestedLineId)?.id ?? firstFactoryLineId(factory);
  const intakes = useFactoryIntakes(organizationId, factoryId);
  const handlers = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const { view, setView } = useLineBoardColumnColorViewPreference();
  const columnColorId = useId();
  const sources = intakeSourcesFromFactoryIntakes(intakes.data ?? []);
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
        {intakes.isLoading || handlers.isLoading ? (
          <p role="status" className="p-4 text-sm text-muted-foreground">
            Loading board options…
          </p>
        ) : null}
        {intakes.isError || handlers.isError ? (
          <div role="alert" className="p-4 text-sm">
            <p>SuperPlane could not load board options.</p>
            <Button
              variant="outline"
              className="mt-2 h-11"
              onClick={() => {
                void intakes.refetch();
                void handlers.refetch();
              }}
            >
              Try again
            </Button>
          </div>
        ) : null}
        {sources.map((intake) => (
          <MoreLink
            key={intake.intakeId}
            href={factoryIntakePath(organizationId, routeSegment, lineId, intake.intakeId)}
            icon={
              <ColumnAutomationGlyph
                automation={{ kind: "intake", iconSrc: intake.source.iconSrc, iconAlt: intake.source.iconAlt }}
                className="size-5"
              />
            }
          >
            <span>
              {lineIntakeListenTitle(intake.source, intake.paused)}
              {!intake.healthy ? <span className="block text-xs text-amber-600">Needs repair</span> : null}
            </span>
          </MoreLink>
        ))}
        {(handlers.data ?? [])
          .filter((handler) => handler.id)
          .map((handler) => (
            <MoreLink
              key={handler.id}
              href={factoryPRFeedbackPath(organizationId, routeSegment, lineId, undefined, handler.id)}
              icon={
                <ColumnAutomationGlyph
                  automation={{ kind: "pr-checks", iconSrc: githubIcon, iconAlt: "GitHub" }}
                  className="size-5"
                />
              }
            >
              <span>
                {prFeedbackListenTitle(handler.source)}
                {handler.healthy === false ? <span className="block text-xs text-amber-600">Needs repair</span> : null}
              </span>
            </MoreLink>
          ))}
      </div>
    </section>
  );
}
