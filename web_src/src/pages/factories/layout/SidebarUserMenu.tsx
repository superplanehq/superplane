import { Avatar } from "@/components/Avatar/avatar";
import { useTheme } from "@/contexts/useTheme";
import { isThemePreference } from "@/lib/themePreference";
import type { ThemePreference } from "@/lib/themePreference";
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
import { LayoutGrid, LogOut, SunMoon, User as UserIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";

interface SidebarUserMenuProps {
  organizationId: string;
  factoryKey?: string;
  userName: string;
  userEmail?: string;
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
  userEmail,
  userAvatarUrl,
  organizationName,
  defaultOpen = false,
}: SidebarUserMenuProps) {
  const navigate = useNavigate();
  const homeHref = `/${organizationId}`;
  const profileHref = factoryKey
    ? factorySettingsSectionPath(organizationId, factoryKey, "profile")
    : `/${organizationId}/settings/profile`;

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
              initials={initialsFor(userName || "?")}
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
          <div className="px-2 py-1.5">
            <p className="truncate text-[13px] font-medium text-foreground">{userName}</p>
            {userEmail ? <p className="truncate text-[11px] text-muted-foreground">{userEmail}</p> : null}
          </div>
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
