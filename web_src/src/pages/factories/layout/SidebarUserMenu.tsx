import { Avatar } from "@/components/Avatar/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { LogOut, MoreHorizontal, User as UserIcon } from "lucide-react";
import { posthog } from "@/posthog";
import { useNavigate } from "react-router-dom";

interface SidebarUserMenuProps {
  organizationId: string;
  userName: string;
  userEmail?: string;
  userAvatarUrl?: string | null;
  organizationName: string;
}

function initialsFor(name: string): string {
  const parts = name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "");
  if (parts.length === 0) {
    return "?";
  }
  return `${parts[0]}${parts[parts.length - 1] ?? ""}`;
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
      className="flex items-center gap-2 border-t border-slate-950/10 px-2 py-3 dark:border-gray-700/70"
      data-testid="factories-sidebar-user-menu"
    >
      <Avatar
        src={userAvatarUrl ?? undefined}
        initials={initialsFor(userName || "?")}
        alt={userName}
        className="size-7 text-[10px]"
      />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{userName}</p>
        <p className="truncate text-xs text-gray-500 dark:text-gray-400">{organizationName}</p>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label="Open user menu"
            data-testid="factories-sidebar-user-menu-trigger"
            className="flex h-6 w-6 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
          >
            <MoreHorizontal className="h-3.5 w-3.5" aria-hidden />
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
              <div className="px-2 py-1.5 text-[11px] text-gray-500 dark:text-gray-400" aria-hidden>
                {userEmail}
              </div>
            </>
          ) : null}
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
