import { safeExternalUrl } from "@/lib/safeExternalUrl";

export function hrefFromInput(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }

  const candidate = /^[a-z][a-z0-9+.-]*:/i.test(trimmed) ? trimmed : `https://${trimmed}`;
  return safeExternalUrl(candidate) ?? "";
}
