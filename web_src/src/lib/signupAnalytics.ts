const pendingSignupAnalyticsPreferenceKey = "superplane:pending_signup_analytics_preference";
const pendingSignupAnalyticsPreferenceMaxAgeMs = 24 * 60 * 60 * 1000;

export type SignupAnalyticsPreference = {
  email?: string;
  productUpdatesOptIn: boolean;
};

export type SignupAnalyticsResult = "created" | "existing" | null;

type StoredSignupAnalyticsPreference = SignupAnalyticsPreference & {
  createdAt: number;
  confirmed: boolean;
};

type ConsumeSignupAnalyticsPreferenceOptions = {
  accountEmail?: string;
  currentPath: string;
  signupResult?: SignupAnalyticsResult;
};

export function savePendingSignupAnalyticsPreference(preference: SignupAnalyticsPreference) {
  writeSignupAnalyticsPreference({
    ...preference,
    createdAt: Date.now(),
    confirmed: false,
  });
}

export function confirmSignupAnalyticsPreference(preference: SignupAnalyticsPreference) {
  writeSignupAnalyticsPreference({
    ...preference,
    createdAt: Date.now(),
    confirmed: true,
  });
}

export function clearPendingSignupAnalyticsPreference() {
  localStorage.removeItem(pendingSignupAnalyticsPreferenceKey);
}

export function consumePendingSignupAnalyticsPreference({
  accountEmail,
  currentPath,
  signupResult = null,
}: ConsumeSignupAnalyticsPreferenceOptions): SignupAnalyticsPreference | null {
  const preference = readSignupAnalyticsPreference();
  if (!preference) {
    return null;
  }

  if (signupResult === "existing") {
    clearPendingSignupAnalyticsPreference();
    return null;
  }

  if (isExpired(preference)) {
    clearPendingSignupAnalyticsPreference();
    return null;
  }

  if (preference.email && accountEmail && preference.email.toLowerCase() !== accountEmail.toLowerCase()) {
    if (signupResult === "created") {
      clearPendingSignupAnalyticsPreference();
    }

    return null;
  }

  if (!preference.confirmed && currentPath !== "/welcome" && signupResult !== "created") {
    return null;
  }

  clearPendingSignupAnalyticsPreference();
  return {
    email: preference.email,
    productUpdatesOptIn: preference.productUpdatesOptIn,
  };
}

function readSignupAnalyticsPreference(): StoredSignupAnalyticsPreference | null {
  const rawPreference = localStorage.getItem(pendingSignupAnalyticsPreferenceKey);
  if (!rawPreference) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawPreference) as Partial<StoredSignupAnalyticsPreference>;
    if (typeof parsed.productUpdatesOptIn !== "boolean" || typeof parsed.createdAt !== "number") {
      return null;
    }

    return {
      email: typeof parsed.email === "string" ? parsed.email : undefined,
      productUpdatesOptIn: parsed.productUpdatesOptIn,
      createdAt: parsed.createdAt,
      confirmed: parsed.confirmed === true,
    };
  } catch {
    return null;
  }
}

function writeSignupAnalyticsPreference(preference: StoredSignupAnalyticsPreference) {
  localStorage.setItem(pendingSignupAnalyticsPreferenceKey, JSON.stringify(preference));
}

function isExpired(preference: StoredSignupAnalyticsPreference) {
  return Date.now() - preference.createdAt > pendingSignupAnalyticsPreferenceMaxAgeMs;
}
