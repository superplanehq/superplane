import { Monitor, Sun, Moon, Check } from "lucide-react";
import { useTheme, type ThemePreference } from "../../hooks/useTheme";

interface ThemeOption {
  value: ThemePreference;
  label: string;
  icon: React.ReactNode;
  preview: React.ReactNode;
}

function LightPreview() {
  return (
    <svg viewBox="0 0 80 56" className="w-full h-full" aria-hidden="true">
      {/* Light theme preview */}
      <rect width="80" height="56" fill="#ffffff" rx="4" />
      {/* Sidebar */}
      <rect x="0" y="0" width="20" height="56" fill="#f9fafb" rx="4" />
      {/* Header */}
      <rect x="20" y="0" width="60" height="10" fill="#f3f4f6" />
      {/* Content lines */}
      <rect x="26" y="16" width="40" height="4" fill="#e5e7eb" rx="1" />
      <rect x="26" y="24" width="32" height="4" fill="#e5e7eb" rx="1" />
      <rect x="26" y="32" width="36" height="4" fill="#e5e7eb" rx="1" />
      {/* Sidebar items */}
      <rect x="4" y="14" width="12" height="3" fill="#d1d5db" rx="1" />
      <rect x="4" y="20" width="10" height="3" fill="#d1d5db" rx="1" />
      <rect x="4" y="26" width="11" height="3" fill="#d1d5db" rx="1" />
    </svg>
  );
}

function DarkPreview() {
  return (
    <svg viewBox="0 0 80 56" className="w-full h-full" aria-hidden="true">
      {/* Dark theme preview */}
      <rect width="80" height="56" fill="#1f2937" rx="4" />
      {/* Sidebar */}
      <rect x="0" y="0" width="20" height="56" fill="#111827" rx="4" />
      {/* Header */}
      <rect x="20" y="0" width="60" height="10" fill="#374151" />
      {/* Content lines */}
      <rect x="26" y="16" width="40" height="4" fill="#4b5563" rx="1" />
      <rect x="26" y="24" width="32" height="4" fill="#4b5563" rx="1" />
      <rect x="26" y="32" width="36" height="4" fill="#4b5563" rx="1" />
      {/* Sidebar items */}
      <rect x="4" y="14" width="12" height="3" fill="#6b7280" rx="1" />
      <rect x="4" y="20" width="10" height="3" fill="#6b7280" rx="1" />
      <rect x="4" y="26" width="11" height="3" fill="#6b7280" rx="1" />
    </svg>
  );
}

function SystemPreview() {
  return (
    <svg viewBox="0 0 80 56" className="w-full h-full" aria-hidden="true">
      {/* Split preview - left light, right dark */}
      <defs>
        <clipPath id="leftHalf">
          <rect x="0" y="0" width="40" height="56" />
        </clipPath>
        <clipPath id="rightHalf">
          <rect x="40" y="0" width="40" height="56" />
        </clipPath>
      </defs>
      {/* Light half */}
      <g clipPath="url(#leftHalf)">
        <rect width="80" height="56" fill="#ffffff" rx="4" />
        <rect x="0" y="0" width="20" height="56" fill="#f9fafb" />
        <rect x="20" y="0" width="60" height="10" fill="#f3f4f6" />
        <rect x="26" y="16" width="40" height="4" fill="#e5e7eb" rx="1" />
        <rect x="26" y="24" width="32" height="4" fill="#e5e7eb" rx="1" />
        <rect x="4" y="14" width="12" height="3" fill="#d1d5db" rx="1" />
        <rect x="4" y="20" width="10" height="3" fill="#d1d5db" rx="1" />
      </g>
      {/* Dark half */}
      <g clipPath="url(#rightHalf)">
        <rect width="80" height="56" fill="#1f2937" rx="4" />
        <rect x="0" y="0" width="20" height="56" fill="#111827" />
        <rect x="20" y="0" width="60" height="10" fill="#374151" />
        <rect x="26" y="16" width="40" height="4" fill="#4b5563" rx="1" />
        <rect x="26" y="24" width="32" height="4" fill="#4b5563" rx="1" />
        <rect x="4" y="14" width="12" height="3" fill="#6b7280" rx="1" />
        <rect x="4" y="20" width="10" height="3" fill="#6b7280" rx="1" />
      </g>
      {/* Diagonal line */}
      <line x1="40" y1="0" x2="40" y2="56" stroke="#9ca3af" strokeWidth="1" />
    </svg>
  );
}

const themeOptions: ThemeOption[] = [
  {
    value: "system",
    label: "System",
    icon: <Monitor className="h-4 w-4" />,
    preview: <SystemPreview />,
  },
  {
    value: "light",
    label: "Light",
    icon: <Sun className="h-4 w-4" />,
    preview: <LightPreview />,
  },
  {
    value: "dark",
    label: "Dark",
    icon: <Moon className="h-4 w-4" />,
    preview: <DarkPreview />,
  },
];

export function ThemeSelector() {
  const { preference, setPreference } = useTheme();

  return (
    <div className="flex gap-4">
      {themeOptions.map((option) => {
        const isSelected = preference === option.value;
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => setPreference(option.value)}
            className={`
              relative flex flex-col items-center gap-2 p-2 rounded-lg border-2 transition-all
              hover:border-gray-400 dark:hover:border-gray-500
              focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900
              ${
                isSelected
                  ? "border-blue-500 bg-blue-50 dark:bg-blue-950"
                  : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
              }
            `}
            aria-pressed={isSelected}
          >
            {/* Preview thumbnail */}
            <div className="w-20 h-14 rounded overflow-hidden border border-gray-200 dark:border-gray-600">
              {option.preview}
            </div>
            {/* Label with icon */}
            <div className="flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300">
              {option.icon}
              <span>{option.label}</span>
            </div>
            {/* Selected checkmark */}
            {isSelected && (
              <div className="absolute top-1 right-1 bg-blue-500 text-white rounded-full p-0.5">
                <Check className="h-3 w-3" />
              </div>
            )}
          </button>
        );
      })}
    </div>
  );
}
