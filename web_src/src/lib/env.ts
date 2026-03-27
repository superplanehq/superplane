export const isCustomComponentsEnabled = () => {
  return import.meta.env.VITE_ENABLE_CUSTOM_COMPONENTS === "true";
};

export const isAgentReplEnabled = () => {
  return import.meta.env.VITE_ENABLE_AGENT_REPL === "true";
};

export const isUsagePageForced = () => {
  return import.meta.env.VITE_FORCE_USAGE_PAGE === "true";
};
