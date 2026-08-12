import { Avatar } from "@/components/Avatar/avatar";
import { useTheme } from "@/contexts/useTheme";
import { cn } from "@/lib/utils";
import { posthog } from "@/posthog";
import type { ThemePreference } from "@/lib/themePreference";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { LogOut, Monitor, Moon, MoreHorizontal, Sun, User as UserIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useNavigate } from "react-router";

interface SidebarUserMenuProps {
  organizationId: string;
  userName: string;
  userEmail?: string;
  userAvatarUrl?: string | null;
  organizationName: string;
}

const THEME_OPTIONS: Array<{ value: ThemePreference; label: string; Icon: LucideIcon }> = [
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
  { value: "system", label: "System", Icon: Monitor },
];

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

export function SidebarUserMenu({
  organizationId,
  userName,
  userEmail,
  userAvatarUrl,
  organizationName,
}: SidebarUserMenuProps) {
  const navigate = useNavigate();
  const homeHref = `/${organizationId}`;
  const profileHref = `/${organizationId}/settings/profile`;

  const handleSignOut = () => {
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <div
      className="flex items-center gap-2 border-t border-sidebar-border px-2 py-3"
      data-testid="factories-sidebar-user-menu"
    >
      <Avatar
        src={userAvatarUrl ?? undefined}
        initials={initialsFor(userName || "?")}
        alt={userName}
        className="size-7 text-[10px]"
      />
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] tracking-[-0.01em] text-foreground">{userName}</p>
        <p className="truncate text-[12px] text-muted-foreground">{organizationName}</p>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label="Open user menu"
            data-testid="factories-sidebar-user-menu-trigger"
            className="flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
          >
            <MoreHorizontal className="size-3.5" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuItem onClick={() => navigate(homeHref)} data-testid="factories-sidebar-back-to-apps">
            <UserIcon className="h-3.5 w-3.5" aria-hidden />
            Back to Apps
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate(profileHref)}>
            <UserIcon className="h-3.5 w-3.5" aria-hidden />
            Profile
          </DropdownMenuItem>
          {userEmail ? (
            <>
              <DropdownMenuSeparator />
              <div className="px-2 py-1.5 text-[11px] text-muted-foreground" aria-hidden>
                {userEmail}
              </div>
            </>
          ) : null}
          <DropdownMenuSeparator />
          <ThemeMenuSection />
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleSignOut}>
            <LogOut className="h-3.5 w-3.5" aria-hidden />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

/**
 * Segmented theme picker (Light / Dark / System) rendered inline in the user
 * menu. Rendered outside the DropdownMenuItem primitive so clicking a theme
 * option doesn't dismiss the menu — that lets the user preview each mode
 * without reopening the menu each time.
 */
function ThemeMenuSection() {
  const { preference, setPreference } = useTheme();

  return (
    <div
      className="px-2 pt-1 pb-2"
      onPointerDown={(event) => {
        // Radix dismisses the menu on any pointerdown inside DropdownMenuContent
        // that isn't handled by a menu primitive; blocking the event here keeps
        // the menu open while the user cycles through themes.
        event.stopPropagation();
      }}
      data-testid="factories-sidebar-theme-picker"
    >
      <DropdownMenuLabel className="px-0 pt-0 pb-1.5 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
        Theme
      </DropdownMenuLabel>
      <div role="radiogroup" aria-label="Theme" className="flex gap-1 rounded-md bg-muted p-1">
        {THEME_OPTIONS.map(({ value, label, Icon }) => {
          const isActive = preference === value;
          return (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={isActive}
              aria-label={label}
              title={label}
              onClick={() => setPreference(value)}
              className={cn(
                "flex flex-1 items-center justify-center rounded-sm px-2 py-1 transition-colors",
                isActive ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
              )}
              data-testid={`factories-sidebar-theme-${value}`}
            >
              <Icon className="size-3.5" aria-hidden />
            </button>
          );
        })}
      </div>
    </div>
  );
}
