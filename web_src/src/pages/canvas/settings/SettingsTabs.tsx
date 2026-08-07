import { Link, useLocation } from "react-router-dom";
import { appSettingsPath, appSecretsPath } from "@/lib/appPaths";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

interface SettingsTabsProps {
  organizationId: string;
  appId: string;
}

/** Simple tab bar for switching between canvas ("app") settings sections. */
export function SettingsTabs({ organizationId, appId }: SettingsTabsProps) {
  const location = useLocation();
  const generalPath = appSettingsPath(organizationId, appId);
  const secretsPath = appSecretsPath(organizationId, appId);
  const isSecretsActive = location.pathname.startsWith(secretsPath);

  const tabs = [
    { label: "General", href: generalPath, active: !isSecretsActive },
    { label: "Secrets", href: secretsPath, active: isSecretsActive },
  ];

  return (
    <div className={cn("flex gap-4 border-b border-slate-200 px-4", appDarkModeClasses.sidebarEdge)}>
      {tabs.map((tab) => (
        <Link
          key={tab.href}
          to={tab.href}
          className={cn(
            "border-b-2 px-1 py-2 text-sm font-medium transition-colors",
            tab.active
              ? "border-sky-500 text-slate-900 dark:text-gray-100"
              : "border-transparent text-slate-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100",
          )}
          data-testid={`canvas-settings-tab-${tab.label.toLowerCase()}`}
        >
          {tab.label}
        </Link>
      ))}
    </div>
  );
}
