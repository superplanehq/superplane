export const isCustomComponentsEnabled = () => {
  return import.meta.env.VITE_ENABLE_CUSTOM_COMPONENTS === "true";
};

export const isAgentReplEnabled = () => {
  return import.meta.env.VITE_ENABLE_AGENT_REPL === "true";
};

export const getAgentReplWebUrl = () => {
  return (import.meta.env.VITE_AGENT_REPL_WEB_URL as string | undefined)?.trim() || "http://localhost:8090";
};
