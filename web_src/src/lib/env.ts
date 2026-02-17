export const isCustomComponentsEnabled = () => {
  return import.meta.env.VITE_ENABLE_CUSTOM_COMPONENTS === "true";
};
