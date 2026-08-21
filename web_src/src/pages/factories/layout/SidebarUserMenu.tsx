import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { Avatar } from "@/components/Avatar/avatar";
import { useTheme } from "@/contexts/useTheme";
import { isThemePreference } from "@/lib/themePreference";
import type { ThemePreference } from "@/lib/themePreference";
import { cn } from "@/lib/utils";
import { posthog } from "@/posthog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import {
  ArrowRightLeft,
  Building2,
  Check,
  LayoutGrid,
  LogOut,
  Plus,
  Settings,
  SunMoon,
  User as UserIcon,
} from "lucide-react";
import { useNavigate } from "react-router";
import { factorySettingsSectionPath, organizationSettingsSectionPath } from "../lib/factoryPagePaths";

interface SidebarUserMenuProps {
  organizationId: string;
  factoryKey?: string;
  userName: string;
  userAvatarUrl?: string | null;
  organizationName: string;
  defaultOpen?: boolean;
}

const THEME_OPTIONS: Array<{ value: ThemePreference; label: string }> = [
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
  { value: "system", label: "System" },
];

const MENU_ITEM_CLASS = "py-1 text-[13px] [&>svg]:size-3.5";

function initialsFor(name: string): string {
  const parts = name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "");
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0];
  }
  return `${parts[0]}${parts[parts.length - 1]}`;
}

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
}: SidebarUserMenuProps) {
  const navigate = useNavigate();
  const homeHref = `/${organizationId}`;
  const profileHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "profile")
    : `/${organizationId}/settings/profile`;
  const organizationHref = factoryKey
    ? organizationSettingsSectionPath(organizationId, factoryKey, "general")
    : `/${organizationId}/settings/general`;

  const handleSignOut = () => {
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <div className="border-t border-sidebar-border p-2" data-testid="factories-sidebar-user-menu">
      <DropdownMenu defaultOpen={defaultOpen}>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            data-testid="factories-sidebar-user-menu-trigger"
            className="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-left hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none data-[state=open]:bg-sidebar-accent"
          >
            <Avatar
              src={userAvatarUrl ?? undefined}
              initials={userAvatarUrl ? undefined : initialsFor(userName || "?")}
              alt=""
              className="size-7 text-[10px]"
            />
            <div className="min-w-0 flex-1">
              <p className="truncate text-[13px] tracking-[-0.01em] text-foreground">{userName}</p>
              <p className="truncate text-[12px] text-muted-foreground">{organizationName}</p>
            </div>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          side="top"
          align="start"
          sideOffset={8}
          className="w-(--radix-popper-anchor-width) min-w-56"
        >
          <OrganizationMenuHeader
            organizationId={organizationId}
            organizationName={organizationName}
            organizationHref={organizationHref}
          />
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            onClick={() => navigate(homeHref)}
            data-testid="factories-sidebar-back-to-apps"
          >
            <LayoutGrid aria-hidden />
            Back to Apps
          </DropdownMenuItem>
          <DropdownMenuItem
            className={MENU_ITEM_CLASS}
            onClick={() => navigate(profileHref)}
            data-testid="factories-sidebar-profile"
          >
            <UserIcon aria-hidden />
            Profile
          </DropdownMenuItem>
          <AppearanceMenuItem />
          <DropdownMenuSeparator />
          <DropdownMenuItem className={MENU_ITEM_CLASS} onClick={handleSignOut}>
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
      <OrganizationSwitchSub currentOrganizationId={organizationId} />
    </div>
  );
}

function OrganizationSwitchSub({ currentOrganizationId }: { currentOrganizationId: string }) {
  const navigate = useNavigate();
  const { data: organizations = [] } = useAccountOrganizations();

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
        <DropdownMenuSubContent className="w-64" data-testid="factories-sidebar-organization-switch-menu">
          <DropdownMenuLabel>Switch organization</DropdownMenuLabel>
          {organizations.map((organization) => {
            const isCurrent = organization.id === currentOrganizationId;
            return (
              <DropdownMenuItem
                key={organization.id}
                onClick={() => {
                  if (!isCurrent) {
                    navigate(`/${organization.id}`);
                  }
                }}
                data-testid={`factories-sidebar-organization-option-${organization.id}`}
              >
                <Building2 className="h-3.5 w-3.5" aria-hidden />
                <span className="truncate">{organization.name}</span>
                {isCurrent ? <Check className="ml-auto h-3.5 w-3.5" aria-hidden /> : null}
              </DropdownMenuItem>
            );
          })}
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => navigate("/create")} data-testid="factories-sidebar-organization-create">
            <Plus className="h-3.5 w-3.5" aria-hidden />
            Create new organization
          </DropdownMenuItem>
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
