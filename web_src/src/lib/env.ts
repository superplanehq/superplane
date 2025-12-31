export const isCustomComponentsEnabled = () => {
  return import.meta.env.VITE_ENABLE_CUSTOM_COMPONENTS === "true";
};

export const isRBACEnabled = () => {
  return import.meta.env.VITE_RBAC_ENABLED === "true";
};
