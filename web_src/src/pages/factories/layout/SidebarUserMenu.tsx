import { Avatar } from "@/components/Avatar/avatar";
import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { useAccount } from "@/contexts/useAccount";
import { useTheme } from "@/contexts/useTheme";
import { isThemePreference } from "@/lib/themePreference";
import type { ThemePreference } from "@/lib/themePreference";
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
import { ArrowRightLeft, LogOut, Settings, Shield, SunMoon, User as UserIcon, XCircle } from "lucide-react";
import { useNavigate } from "react-router";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, initialsForName } from "./factoriesRail";

export interface SidebarUserMenuProps {
  organizationId: string;
  factoryKey?: string;
  userName: string;
  userEmail?: string;
  userAvatarUrl?: string | null;
  organizationName: string;
  defaultOpen?: boolean;
  /**
   * `sidebar` (default) is the factory rail: a bordered row that opens the
   * menu to the right. `floating` drops the border for placement anywhere
   * else on screen, such as the onboarding shell's bottom-left corner, and
   * skips Appearance to keep the menu short.
   */
  variant?: "sidebar" | "floating";
  /**
   * Shown above Sign out when set. Onboarding passes this to let a user who
   * has somewhere else to go leave setup, deleting the placeholder
   * workspace. Unset elsewhere, and whenever onboarding has nowhere else to
   * send the user — Sign out still works either way.
   */
  onQuitOnboarding?: () => void;
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
  userEmail,
  userAvatarUrl,
  organizationName,
  defaultOpen = false,
  variant = "sidebar",
  onQuitOnboarding,
}: SidebarUserMenuProps) {
  const navigate = useNavigate();
  const { account } = useAccount();
  const profileHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "account", "general")
    : `/${organizationId}/settings/profile`;
  const organizationHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "organization", "general")
    : `/${organizationId}/settings/general`;
  const identityLabel = userEmail
    ? `${userName} (${userEmail}), ${organizationName}`
    : `${userName}, ${organizationName}`;

  const handleSignOut = () => {
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <div
      className={cn("flex justify-center", variant === "sidebar" && "border-t border-sidebar-border p-1.5")}
      data-testid="factories-sidebar-user-menu"
    >
      <DropdownMenu defaultOpen={defaultOpen}>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            data-testid="factories-sidebar-user-menu-trigger"
            aria-label={identityLabel}
            title={identityLabel}
            className={cn(
              factoriesRailControlClassName,
              variant === "floating" && "bg-background shadow-sm",
              "data-[state=open]:bg-sidebar-accent",
            )}
          >
            <Avatar
              src={userAvatarUrl ?? undefined}
              initials={userAvatarUrl ? undefined : initialsForName(userName || "?")}
              alt=""
              className="size-7 text-[10px]"
            />
            <span className="sr-only">{identityLabel}</span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="right" align="end" sideOffset={8} className="min-w-56">
          <OrganizationMenuHeader
            organizationId={organizationId}
            organizationName={organizationName}
            organizationHref={organizationHref}
          />
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            onClick={() => navigate(profileHref)}
            data-testid="factories-sidebar-profile"
          >
            <UserIcon aria-hidden />
            Profile
          </DropdownMenuItem>
          {account?.installation_admin ? (
            <DropdownMenuItem
              className={MENU_ITEM_CLASS}
              onClick={() => navigate("/admin")}
              data-testid="factories-sidebar-installation-admin"
            >
              <Shield aria-hidden />
              Installation Admin
            </DropdownMenuItem>
          ) : null}
          {variant === "sidebar" ? <AppearanceMenuItem /> : null}
          <DropdownMenuSeparator />
          {onQuitOnboarding ? (
            <DropdownMenuItem
              className={MENU_ITEM_CLASS}
              onClick={onQuitOnboarding}
              data-testid="factories-sidebar-quit-onboarding"
            >
              <XCircle aria-hidden />
              Quit onboarding
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            onClick={handleSignOut}
            data-testid="factories-sidebar-sign-out"
          >
            <LogOut aria-hidden />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

const HEADER_ICON_CLASS =
  "flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

function OrganizationMenuHeader({
  organizationId,
  organizationName,
  organizationHref,
}: {
  organizationId: string;
  organizationName: string;
  organizationHref: string;
}) {
  const navigate = useNavigate();

  return (
    <div className="flex items-center gap-0.5 px-1 py-1" data-testid="factories-sidebar-organization">
      <p
        className="min-w-0 flex-1 truncate px-2 py-1 text-[13px] font-medium tracking-[-0.01em] text-foreground"
        data-testid="factories-sidebar-organization-name"
      >
        {organizationName}
      </p>
      <DropdownMenuItem
        aria-label="Organization settings"
        data-testid="factories-sidebar-organization-settings-link"
        className={cn(HEADER_ICON_CLASS, "cursor-pointer p-0")}
        onSelect={() => navigate(organizationHref)}
      >
        <Settings className="size-3.5" aria-hidden />
      </DropdownMenuItem>
      <OrganizationSwitchSub currentOrganizationRouteId={organizationId} />
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
