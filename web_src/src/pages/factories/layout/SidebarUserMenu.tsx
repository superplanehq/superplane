import { Avatar } from "@/components/Avatar/avatar";
import { FeedbackDialog } from "@/components/FeedbackDialog";
import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { useAccount } from "@/contexts/useAccount";
import { useTheme } from "@/contexts/useTheme";
import { isThemePreference } from "@/lib/themePreference";
import type { ThemePreference } from "@/lib/themePreference";
import type { FeedbackCategory } from "@/lib/submitFeedback";
import { cn } from "@/lib/utils";
import { posthog } from "@/posthog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { ArrowRightLeft, Bug, LogOut, MessageSquare, Settings, Shield, SunMoon, User as UserIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, initialsForName } from "./factoriesRail";

interface SidebarUserMenuProps {
  organizationId: string;
  factoryKey?: string;
  userName: string;
  userAvatarUrl?: string | null;
  organizationName: string;
  defaultOpen?: boolean;
  planLabel?: string;
}

const THEME_OPTIONS: Array<{ value: ThemePreference; label: string }> = [
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
  { value: "system", label: "System" },
];

const MENU_ITEM_CLASS = "py-1 text-[13px] [&>svg]:size-3.5";

function labelForTheme(preference: ThemePreference): string {
  return THEME_OPTIONS.find((option) => option.value === preference)?.label ?? "System";
}

export function SidebarUserMenu({
  organizationId,
  factoryKey,
  userName,
  userAvatarUrl,
  organizationName,
  defaultOpen = false,
  planLabel,
}: SidebarUserMenuProps) {
  const { account } = useAccount();
  const profileHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "account", "general")
    : `/${organizationId}/settings/profile`;
  const organizationHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "organization", "general")
    : `/${organizationId}/settings/general`;
  const triggerLabel = `${userName}, ${organizationName}`;
  const [isFeedbackOpen, setIsFeedbackOpen] = useState(false);
  const [feedbackCategory, setFeedbackCategory] = useState<FeedbackCategory | undefined>();

  const openFeedback = (category: FeedbackCategory) => {
    setFeedbackCategory(category);
    setIsFeedbackOpen(true);
  };

  const handleSignOut = () => {
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <div
      className="flex flex-col items-center border-t border-sidebar-border px-1 py-1.5"
      data-testid="factories-sidebar-user-menu"
    >
      <DropdownMenu defaultOpen={defaultOpen}>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            data-testid="factories-sidebar-user-menu-trigger"
            aria-label={triggerLabel}
            title={triggerLabel}
            className="group flex flex-col items-center gap-0.5 rounded-md focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          >
            <span
              className={cn(
                factoriesRailControlClassName,
                "pointer-events-none group-hover:bg-sidebar-accent group-hover:text-foreground group-data-[state=open]:bg-sidebar-accent",
              )}
            >
              <Avatar
                src={userAvatarUrl ?? undefined}
                initials={userAvatarUrl ? undefined : initialsForName(userName || "?")}
                alt=""
                className="size-7 text-[10px]"
              />
            </span>
            <span className="sr-only">{triggerLabel}</span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="right" align="end" sideOffset={8} className="min-w-56">
          <OrganizationMenuHeader
            organizationId={organizationId}
            organizationName={organizationName}
            organizationHref={organizationHref}
            planLabel={planLabel}
          />
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild className={MENU_ITEM_CLASS} data-testid="factories-sidebar-profile">
            <Link to={profileHref}>
              <UserIcon aria-hidden />
              Profile
            </Link>
          </DropdownMenuItem>
          {account?.installation_admin ? (
            <DropdownMenuItem asChild className={MENU_ITEM_CLASS} data-testid="factories-sidebar-installation-admin">
              <Link to="/admin">
                <Shield aria-hidden />
                Installation Admin
              </Link>
            </DropdownMenuItem>
          ) : null}
          <AppearanceMenuItem />
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            data-testid="factories-sidebar-report-issue"
            onClick={() => openFeedback("bug")}
          >
            <Bug aria-hidden />
            Report issue
          </DropdownMenuItem>
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            data-testid="factories-sidebar-send-feedback"
            onClick={() => openFeedback("other")}
          >
            <MessageSquare aria-hidden />
            Send feedback
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem className={MENU_ITEM_CLASS} onClick={handleSignOut}>
            <LogOut aria-hidden />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <FeedbackDialog
        open={isFeedbackOpen}
        onOpenChange={setIsFeedbackOpen}
        organizationId={organizationId}
        initialCategory={feedbackCategory}
      />
    </div>
  );
}

const HEADER_ICON_CLASS =
  "flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

function OrganizationMenuHeader({
  organizationId,
  organizationName,
  organizationHref,
  planLabel,
}: {
  organizationId: string;
  organizationName: string;
  organizationHref: string;
  planLabel?: string;
}) {
  return (
    <div className="px-1 py-1" data-testid="factories-sidebar-organization">
      <div className="flex items-center gap-0.5">
        <p
          className="min-w-0 flex-1 truncate px-2 py-1 text-[13px] font-medium tracking-[-0.01em] text-foreground"
          data-testid="factories-sidebar-organization-name"
        >
          {organizationName}
        </p>
        <DropdownMenuItem
          asChild
          aria-label="Organization settings"
          data-testid="factories-sidebar-organization-settings-link"
          className={cn(HEADER_ICON_CLASS, "cursor-pointer p-0")}
        >
          <Link to={organizationHref}>
            <Settings className="size-3.5" aria-hidden />
          </Link>
        </DropdownMenuItem>
        <OrganizationSwitchSub currentOrganizationRouteId={organizationId} />
      </div>
      {planLabel ? (
        <p className="px-2 text-[11px] text-muted-foreground" data-testid="factories-sidebar-plan-status">
          {planLabel}
        </p>
      ) : null}
    </div>
  );
}

function OrganizationSwitchSub({ currentOrganizationRouteId }: { currentOrganizationRouteId: string }) {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger
        aria-label="Switch organization"
        data-testid="factories-sidebar-organization-switch"
        className={cn(HEADER_ICON_CLASS, "cursor-pointer p-0 [&_svg]:size-3.5 [&>svg:last-child]:hidden")}
      >
        <ArrowRightLeft className="size-3.5" aria-hidden />
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent
          className="max-h-[var(--radix-dropdown-menu-content-available-height)] w-64 overflow-y-auto"
          data-testid="factories-sidebar-organization-switch-menu"
        >
          <OrganizationSwitchMenu
            currentOrganizationRouteId={currentOrganizationRouteId}
            testIdPrefix="factories-sidebar"
          />
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}

function AppearanceMenuItem() {
  const { preference, setPreference } = useTheme();

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger
        className="py-1 text-[13px] [&_svg]:size-3.5 [&>svg:last-child]:ml-0"
        data-testid="factories-sidebar-appearance"
      >
        <SunMoon aria-hidden />
        Appearance
        <span className="ml-auto text-[11px] text-muted-foreground">{labelForTheme(preference)}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent>
          <DropdownMenuRadioGroup
            value={preference}
            onValueChange={(value) => {
              if (isThemePreference(value)) {
                setPreference(value);
              }
            }}
          >
            {THEME_OPTIONS.map(({ value, label }) => (
              <DropdownMenuRadioItem
                key={value}
                value={value}
                className="py-1 text-[13px]"
                data-testid={`factories-sidebar-theme-${value}`}
              >
                {label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
