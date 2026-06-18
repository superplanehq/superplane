export const isUsagePageForced = () => {
  return import.meta.env.VITE_FORCE_USAGE_PAGE === "true";
};
