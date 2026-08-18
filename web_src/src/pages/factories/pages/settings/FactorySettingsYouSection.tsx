import { Avatar } from "@/components/Avatar/avatar";
import { ThemePreferenceControl } from "@/components/ThemePreferenceControl";
import { cn } from "@/lib/utils";
import { posthog } from "@/posthog";
import { LayoutGrid, LogOut } from "lucide-react";
import { Link, NavLink } from "react-router";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { FACTORY_SETTINGS_NAV_ITEMS } from "./settingsNavItems";

interface FactorySettingsYouSectionProps {
  organizationId: string;
  factoryKey: string;
  userName: string;
  userEmail?: string;
  userAvatarUrl?: string | null;
}

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

function youNavLinkClass(isActive: boolean) {
  return cn(
    "group flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
    isActive && "bg-sidebar-accent font-medium text-foreground",
  );
}

export function FactorySettingsYouSection({
  organizationId,
  factoryKey,
  userName,
  userEmail,
  userAvatarUrl,
}: FactorySettingsYouSectionProps) {
  const youItems = FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.group === "you");

  const handleSignOut = () => {
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <div className="border-t border-sidebar-border px-2 py-3" data-testid="factory-settings-you-section">
      <div className="flex items-center gap-2.5 px-2.5">
        <Avatar
          src={userAvatarUrl ?? undefined}
          initials={initialsFor(userName || "?")}
          alt=""
          className="size-8 text-[11px]"
        />
        <div className="min-w-0">
          <p className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{userName}</p>
          {userEmail ? <p className="truncate text-[12px] text-muted-foreground">{userEmail}</p> : null}
        </div>
      </div>
      <div className="mt-3 flex flex-col gap-0.5">
        {youItems.map((item) => {
          const Icon = item.Icon;
          return (
            <NavLink
              key={item.id}
              to={factorySettingsSectionPath(organizationId, factoryKey, item.id)}
              className={({ isActive }) => youNavLinkClass(isActive)}
              data-testid={`factory-settings-nav-${item.id}`}
              aria-label={item.id === "profile" ? "Profile" : undefined}
            >
              <Icon className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
              <span>{item.label}</span>
            </NavLink>
          );
        })}
        <Link to={`/${organizationId}`} className={youNavLinkClass(false)} data-testid="factory-settings-back-to-apps">
          <LayoutGrid className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
          <span>Back to Apps</span>
        </Link>
        <button
          type="button"
          onClick={handleSignOut}
          className={youNavLinkClass(false)}
          data-testid="factory-settings-sign-out"
        >
          <LogOut className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
          <span>Sign Out</span>
        </button>
        <ThemePreferenceControl variant="workspace" />
      </div>
    </div>
  );
}
