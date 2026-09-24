import sentryIcon from "@/assets/icons/integrations/sentry.svg";

const LOGO_DARK_INVERT_CLASS = "dark:brightness-0 dark:invert";

export const LOGO_DARK_INVERT_SOURCES = new Set<string>([sentryIcon]);

export function logoDarkInvertClass(src?: string): string | undefined {
  if (!src || !LOGO_DARK_INVERT_SOURCES.has(src)) {
    return undefined;
  }
  return LOGO_DARK_INVERT_CLASS;
}
