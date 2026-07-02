const UTM_STORAGE_KEY = "superplane.initial_utm";
const UTM_COOKIE_NAME = "superplane_initial_utm";
const UTM_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 180;

const UTM_KEYS = ["utm_source", "utm_campaign", "utm_medium", "utm_content"] as const;

export type UTMKey = (typeof UTM_KEYS)[number];
export type UTMAttribution = Partial<Record<UTMKey, string>>;

type PostHogPeopleClient = {
  people?: {
    set_once?: (properties: Record<string, string>) => void;
  };
};

export const getUtmAttributionFromSearch = (search: string): UTMAttribution => {
  const params = new URLSearchParams(search);
  return UTM_KEYS.reduce<UTMAttribution>((values, key) => {
    const value = params.get(key)?.trim();
    if (value) {
      values[key] = value;
    }

    return values;
  }, {});
};

export const getStoredUtmAttribution = (): UTMAttribution => {
  const fromLocalStorage = readLocalStorageAttribution();
  if (hasUtmAttribution(fromLocalStorage)) {
    return fromLocalStorage;
  }

  const fromCookie = readCookieAttribution();
  if (hasUtmAttribution(fromCookie)) {
    writeLocalStorageAttribution(fromCookie);
    return fromCookie;
  }

  return {};
};

export const initializeUtmAttribution = (posthog: PostHogPeopleClient) => {
  const currentAttribution = getUtmAttributionFromSearch(window.location.search);
  const storedAttribution = getStoredUtmAttribution();
  const attribution = hasUtmAttribution(storedAttribution) ? storedAttribution : currentAttribution;

  if (!hasUtmAttribution(attribution)) {
    return;
  }

  writeLocalStorageAttribution(attribution);
  writeCookieAttribution(attribution);
  posthog.people?.set_once?.(toInitialUtmPersonProperties(attribution));
};

export const getUtmEventProperties = (): UTMAttribution => getStoredUtmAttribution();

export const getUtmCookieDomain = (hostname: string) => {
  if (hostname === "superplane.com" || hostname.endsWith(".superplane.com")) {
    return ".superplane.com";
  }

  if (hostname === "superplane.io" || hostname.endsWith(".superplane.io")) {
    return ".superplane.io";
  }

  return undefined;
};

const toInitialUtmPersonProperties = (attribution: UTMAttribution) =>
  UTM_KEYS.reduce<Record<string, string>>((properties, key) => {
    const value = attribution[key];
    if (value) {
      properties[`$initial_${key}`] = value;
    }

    return properties;
  }, {});

const hasUtmAttribution = (attribution: UTMAttribution) => Object.keys(attribution).length > 0;

const readLocalStorageAttribution = (): UTMAttribution => {
  try {
    const rawValue = window.localStorage.getItem(UTM_STORAGE_KEY);
    return parseAttribution(rawValue);
  } catch {
    return {};
  }
};

const writeLocalStorageAttribution = (attribution: UTMAttribution) => {
  try {
    window.localStorage.setItem(UTM_STORAGE_KEY, JSON.stringify(attribution));
  } catch {
    // Attribution is best effort; blocked storage should not affect auth.
  }
};

const readCookieAttribution = (): UTMAttribution => {
  const prefix = `${UTM_COOKIE_NAME}=`;
  const rawValue = document.cookie
    .split(";")
    .map((cookie) => cookie.trim())
    .find((cookie) => cookie.startsWith(prefix))
    ?.slice(prefix.length);

  if (!rawValue) {
    return {};
  }

  try {
    return parseAttribution(decodeURIComponent(rawValue));
  } catch {
    return {};
  }
};

const writeCookieAttribution = (attribution: UTMAttribution) => {
  const encodedValue = encodeURIComponent(JSON.stringify(attribution));
  const cookieParts = [
    `${UTM_COOKIE_NAME}=${encodedValue}`,
    "Path=/",
    `Max-Age=${UTM_COOKIE_MAX_AGE_SECONDS}`,
    "SameSite=Lax",
  ];

  const domain = getUtmCookieDomain(window.location.hostname);
  if (domain) {
    cookieParts.push(`Domain=${domain}`);
  }

  if (window.location.protocol === "https:") {
    cookieParts.push("Secure");
  }

  document.cookie = cookieParts.join("; ");
};

const parseAttribution = (rawValue: string | null | undefined): UTMAttribution => {
  if (!rawValue) {
    return {};
  }

  try {
    const parsedValue = JSON.parse(rawValue) as Record<string, unknown>;
    return UTM_KEYS.reduce<UTMAttribution>((values, key) => {
      const value = parsedValue[key];
      if (typeof value === "string" && value.trim()) {
        values[key] = value.trim();
      }

      return values;
    }, {});
  } catch {
    return {};
  }
};
