type SuperplaneWindow = Window & {
  SUPERPLANE_AGENT_ENABLED?: boolean;
};

export function isAgentEnabled(): boolean {
  return (window as SuperplaneWindow).SUPERPLANE_AGENT_ENABLED ?? false;
}

export const isUsagePageForced = () => {
  return import.meta.env.VITE_FORCE_USAGE_PAGE === "true";
};
