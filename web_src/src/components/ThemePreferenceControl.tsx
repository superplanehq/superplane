import { cn } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import type { ThemePreference } from "@/lib/themePreference";
import { useTheme } from "@/contexts/useTheme";
import { Monitor, Moon, Sun } from "lucide-react";

const OPTIONS: Array<{ value: ThemePreference; label: string; Icon: typeof Sun }> = [
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
  { value: "system", label: "System", Icon: Monitor },
];

export function ThemePreferenceControl() {
  const { preference, setPreference } = useTheme();

  return (
    <div className={cn("-mx-4 mt-2 border-t px-4 pt-4 pb-3", appDarkModeClasses.sidebarDivider)}>
      <div className="flex h-7 gap-0.5 rounded-full bg-gray-100 p-0.5 dark:bg-gray-800">
        {OPTIONS.map(({ value, label, Icon }) => {
          const isActive = preference === value;

          return (
            <button
              key={value}
              type="button"
              aria-pressed={isActive}
              onClick={() => setPreference(value)}
              className={cn(
                "flex flex-1 items-center justify-center gap-1 rounded-full px-1.5 text-xs font-medium transition-colors",
                isActive
                  ? "my-px bg-white text-gray-800 shadow-sm dark:bg-gray-700 dark:text-gray-100 dark:shadow-none"
                  : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200",
              )}
            >
              <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden />
              <span>{label}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
