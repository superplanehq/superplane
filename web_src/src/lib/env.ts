export const isCustomComponentsEnabled = () => {
  return import.meta.env.VITE_ENABLE_CUSTOM_COMPONENTS === "true";
};

export const isAgentReplEnabled = () => {
  return import.meta.env.VITE_ENABLE_AGENT_REPL === "true";
};

export const getAgentUrl = () => {
  return (import.meta.env.VITE_AGENT_URL as string | undefined)?.trim() || "http://localhost:8090";
};
