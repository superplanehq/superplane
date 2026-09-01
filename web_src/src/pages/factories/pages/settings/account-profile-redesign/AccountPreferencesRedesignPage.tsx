import { Label } from "@/components/ui/label";
import type { ThemePreference } from "@/lib/themePreference";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignProfile } from "./accountProfileRedesignMocks";
import { SettingsChoice } from "./accountProfileRedesignParts";

const THEME_CHOICES: Array<{ value: ThemePreference; label: string; description: string }> = [
  { value: "system", label: "System", description: "Match the device appearance." },
  { value: "light", label: "Light", description: "Always use the light theme." },
  { value: "dark", label: "Dark", description: "Always use the dark theme." },
];

const TIMEZONE_CHOICES: Array<{
  value: AccountRedesignProfile["timezone"];
  label: string;
  description: string;
}> = [
  { value: "auto", label: "Automatic", description: "Use the timezone from this device." },
  { value: "America/New_York", label: "New York", description: "Eastern Time (UTC-5 / UTC-4)." },
  { value: "Europe/London", label: "London", description: "Greenwich Mean Time (UTC+0 / UTC+1)." },
  { value: "Asia/Tokyo", label: "Tokyo", description: "Japan Standard Time (UTC+9)." },
];

export function AccountPreferencesRedesignPage({
  theme,
  timezone,
  onThemeChange,
  onTimezoneChange,
}: {
  theme: ThemePreference;
  timezone: AccountRedesignProfile["timezone"];
  onThemeChange: (theme: ThemePreference) => void;
  onTimezoneChange: (timezone: AccountRedesignProfile["timezone"]) => void;
}) {
  return (
    <FactorySettingsPageFrame title="Preferences" subtitle="Choose how SuperPlane looks and shows time.">
      <FactorySettingsCard title="Appearance">
        <div className="space-y-3">
          <div>
            <Label>Theme</Label>
            <p className="mt-0.5 text-[12px] text-muted-foreground">Choose how SuperPlane looks on this device.</p>
          </div>
          <div className="space-y-2" role="radiogroup" aria-label="Theme">
            {THEME_CHOICES.map((choice) => (
              <SettingsChoice
                key={choice.value}
                id={`account-redesign-theme-${choice.value}`}
                label={choice.label}
                description={choice.description}
                checked={theme === choice.value}
                onSelect={() => onThemeChange(choice.value)}
              />
            ))}
          </div>
        </div>
      </FactorySettingsCard>

      <FactorySettingsCard title="Time">
        <div className="space-y-3">
          <div>
            <Label>Timezone</Label>
            <p className="mt-0.5 text-[12px] text-muted-foreground">Used for run times and notification hours.</p>
          </div>
          <div className="space-y-2" role="radiogroup" aria-label="Timezone" data-testid="account-redesign-timezone">
            {TIMEZONE_CHOICES.map((choice) => (
              <SettingsChoice
                key={choice.value}
                id={`account-redesign-tz-${choice.value}`}
                label={choice.label}
                description={choice.description}
                checked={timezone === choice.value}
                onSelect={() => onTimezoneChange(choice.value)}
              />
            ))}
          </div>
        </div>
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}
