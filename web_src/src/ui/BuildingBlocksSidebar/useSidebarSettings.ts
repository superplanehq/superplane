import { useEffect, useState } from "react";

export const BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY = "buildingBlocksSidebarSettings";

interface SidebarSettings {
  showIntegrationSetupStatus: boolean;
  showConnectedIntegrationsOnTop: boolean;
}

const DEFAULTS: SidebarSettings = {
  showIntegrationSetupStatus: true,
  showConnectedIntegrationsOnTop: false,
};

function readSettings(): SidebarSettings {
  if (typeof window === "undefined") {
    return DEFAULTS;
  }

  const raw = window.localStorage.getItem(BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY);
  if (raw === null) {
    return DEFAULTS;
  }

  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return DEFAULTS;
    }

    const obj = parsed as Record<string, unknown>;
    return {
      showIntegrationSetupStatus:
        typeof obj.showIntegrationSetupStatus === "boolean"
          ? obj.showIntegrationSetupStatus
          : DEFAULTS.showIntegrationSetupStatus,
      showConnectedIntegrationsOnTop:
        typeof obj.showConnectedIntegrationsOnTop === "boolean"
          ? obj.showConnectedIntegrationsOnTop
          : DEFAULTS.showConnectedIntegrationsOnTop,
    };
  } catch {
    return DEFAULTS;
  }
}

/**
 * Manages the BuildingBlocksSidebar display settings and persists them
 * to localStorage so they survive remounts and page reloads.
 */
export function useSidebarSettings() {
  const [settings, setSettings] = useState<SidebarSettings>(readSettings);

  useEffect(() => {
    window.localStorage.setItem(BUILDING_BLOCKS_SIDEBAR_SETTINGS_KEY, JSON.stringify(settings));
  }, [settings]);

  return {
    showIntegrationSetupStatus: settings.showIntegrationSetupStatus,
    showConnectedIntegrationsOnTop: settings.showConnectedIntegrationsOnTop,
    setShowIntegrationSetupStatus: (value: boolean) =>
      setSettings((prev) => ({ ...prev, showIntegrationSetupStatus: value })),
    setShowConnectedIntegrationsOnTop: (value: boolean) =>
      setSettings((prev) => ({ ...prev, showConnectedIntegrationsOnTop: value })),
  };
}
