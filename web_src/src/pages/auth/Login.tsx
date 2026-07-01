import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import superplaneLogo from "@/assets/superplane.svg";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAccount } from "../../contexts/useAccount";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import {
  readLastUsedLoginMethod,
  recordLastUsedLoginMethod,
  type LastUsedLoginMethod,
} from "@/lib/lastUsedLoginMethod";
import {
  clearPendingSignupAnalyticsPreference,
  confirmSignupAnalyticsPreference,
  savePendingSignupAnalyticsPreference,
} from "@/lib/signupAnalytics";
import { hasSignupWaitlistConfig } from "@/lib/signupWaitlistConfig";
import { buildMagicLinkVerifyRequest } from "./magicLinkVerifyRequest";
import { getAuthRedirectURL, getWelcomeRedirectPath } from "./authRedirect";
import { SignupWaitlist } from "./SignupWaitlist";
import { getSignupUnavailableReason, type SignupUnavailableReason } from "./signupUnavailableReason";

type AuthConfig = {
  providers: string[];
  passwordLoginEnabled: boolean;
  signupEnabled: boolean;
  signupsBlockedByEnvironment: boolean;
  magicCodeEnabled: boolean;
};

const isValidRedirectPath = (path: string | null): path is string => {
  if (!path || path[0] !== "/") {
    return false;
  }

  if (path.length > 1 && path[1] === "/") {
    return false;
  }

  return true;
};

const getSafeRedirectPath = (rawRedirect: string | null): string | null => {
  if (!rawRedirect) {
    return null;
  }

  try {
    const decoded = decodeURIComponent(rawRedirect);
    return isValidRedirectPath(decoded) ? decoded : null;
  } catch {
    return null;
  }
};

const getProviderLabel = (provider: string) => {
  switch (provider) {
    case "github":
      return "GitHub";
    case "google":
      return "Google";
    default:
      return provider.charAt(0).toUpperCase() + provider.slice(1);
  }
};

const providerAuthPath = (provider: string, redirectQuery: string, isSignupMode: boolean) => {
  const params = new URLSearchParams(redirectQuery ? redirectQuery.slice(1) : "");
  if (isSignupMode) {
    params.set("signup", "true");
  }

  const query = params.toString();
  return query ? `/auth/${provider}?${query}` : `/auth/${provider}`;
};

type MagicCodeStep = "email" | "code";
type AuthMode = "login" | "signup";

interface LoginProps {
  mode?: AuthMode;
}

const getAuthErrorMessage = (authError: string | null, signupUnavailableReason: SignupUnavailableReason) => {
  if (authError === "signup_required") {
    if (signupUnavailableReason === "waitlist") {
      return "No account exists for that provider yet. Join the waitlist to request access.";
    }

    if (signupUnavailableReason === "closed") {
      return "No account exists for that provider yet. Signups are not available right now.";
    }

    return "No account exists for that provider yet. Create an account to continue.";
  }

  return null;
};

const LastUsedHint: React.FC<{ label: string }> = ({ label }) => (
  <p className="mt-2 text-center text-xs text-gray-500">You used {label} to log in last time</p>
);

const ProductUpdatesOptIn: React.FC<{
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}> = ({ checked, onCheckedChange }) => (
  <label htmlFor="signup-product-updates" className="flex cursor-pointer items-start gap-2 text-sm text-gray-700">
    <Checkbox
      id="signup-product-updates"
      checked={checked}
      onChange={(e: React.ChangeEvent<HTMLInputElement>) => onCheckedChange(e.target.checked)}
      className="mt-0.5"
    />
    <span>I want to receive product updates</span>
  </label>
);

const SignupClosedNotice: React.FC = () => (
  <p className="text-left text-sm leading-6 text-gray-600">
    This installation is not accepting new account signups right now. Contact your SuperPlane administrator if you need
    access.
  </p>
);

const saveSignupPreference = (enabled: boolean, productUpdatesOptIn: boolean, email?: string) => {
  if (!enabled) {
    return;
  }

  savePendingSignupAnalyticsPreference({
    email: email?.trim() || undefined,
    productUpdatesOptIn,
  });
};

const clearSignupPreference = (enabled: boolean) => {
  if (enabled) {
    clearPendingSignupAnalyticsPreference();
  }
};

const buildMagicCodeRequestBody = (email: string, signup: boolean, redirectTarget: string) => {
  const formData = new URLSearchParams();
  formData.append("email", email.trim());
  if (signup) {
    formData.append("signup", "true");
  }
  if (redirectTarget) {
    formData.append("redirect", redirectTarget);
  }

  return formData.toString();
};

const buildMagicCodeVerifyBody = (email: string, code: string, signup: boolean, inviteToken: string) => {
  const formData = new URLSearchParams();
  formData.append("email", email.trim());
  formData.append("code", code.trim());
  if (signup) {
    formData.append("signup", "true");
  }
  if (inviteToken) {
    formData.append("invite_token", inviteToken);
  }

  return formData.toString();
};

const buildSignupBody = (name: string, email: string, password: string, inviteToken: string) => {
  const formData = new URLSearchParams();
  formData.append("name", name);
  formData.append("email", email.trim());
  formData.append("password", password);
  if (inviteToken) {
    formData.append("invite_token", inviteToken);
  }

  return formData.toString();
};

const getMagicCodeVerifyError = async (response: Response) => {
  if (response.status === 401) {
    return "Invalid or expired code. Please try again.";
  }

  if (response.status === 403) {
    const errorText = await response.text();
    return errorText || "Sign-up is not allowed.";
  }

  return "Verification failed. Please try again.";
};

const getSignupError = async (response: Response) => {
  if (response.status === 409) {
    return "Account already exists. Please sign in.";
  }

  const errorText = await response.text();
  return errorText || "Signup failed. Please try again.";
};

type SignupValidationFields = {
  canSignup: boolean;
  firstName: string;
  lastName: string;
  email: string;
  password: string;
  confirmPassword: string;
};

const getSignupValidationError = ({
  canSignup,
  firstName,
  lastName,
  email,
  password,
  confirmPassword,
}: SignupValidationFields) => {
  if (!canSignup) {
    return "Signups are currently disabled.";
  }

  if (!firstName.trim() || !lastName.trim()) {
    return "First and last names are required";
  }

  if (!email.trim() || !password || !confirmPassword) {
    return "Email and password are required";
  }

  if (password !== confirmPassword) {
    return "Passwords do not match";
  }

  return null;
};

export const Login: React.FC<LoginProps> = ({ mode = "login" }) => {
  const [authConfig, setAuthConfig] = useState<AuthConfig>({
    providers: [],
    passwordLoginEnabled: false,
    signupEnabled: true,
    signupsBlockedByEnvironment: false,
    magicCodeEnabled: false,
  });
  const [configLoading, setConfigLoading] = useState(true);
  const [configError, setConfigError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [isSignupMode, setIsSignupMode] = useState(mode === "signup");
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [signupFirstName, setSignupFirstName] = useState("");
  const [signupLastName, setSignupLastName] = useState("");
  const [signupEmail, setSignupEmail] = useState("");
  const [signupPassword, setSignupPassword] = useState("");
  const [signupConfirmPassword, setSignupConfirmPassword] = useState("");
  const [signupProductUpdatesOptIn, setSignupProductUpdatesOptIn] = useState(true);

  const [magicCodeStep, setMagicCodeStep] = useState<MagicCodeStep>("email");
  const [magicCodeEmail, setMagicCodeEmail] = useState("");
  const [magicCode, setMagicCode] = useState("");
  const [showPasswordLogin, setShowPasswordLogin] = useState(false);

  const [lastUsedMethod, setLastUsedMethod] = useState<LastUsedLoginMethod | null>(null);

  const [searchParams] = useSearchParams();
  const { account, loading: accountLoading } = useAccount();

  const redirectParam = searchParams.get("redirect");
  const safeRedirect = useMemo(() => getSafeRedirectPath(redirectParam), [redirectParam]);

  const inviteToken = useMemo(() => {
    if (!safeRedirect || !safeRedirect.startsWith("/invite/")) {
      return "";
    }

    const parts = safeRedirect.split("/");
    return parts.length >= 3 ? parts[2] : "";
  }, [safeRedirect]);

  const magicLinkToken = searchParams.get("magic_link_token");
  const redirectTarget = safeRedirect || "";
  const authError = searchParams.get("auth_error");

  const handleRedirectAfterAuth = useCallback(
    async (response: Response, authRedirectURL?: string) => {
      const finalURL = authRedirectURL ?? response.url ?? "/";
      const welcomeRedirectPath = getWelcomeRedirectPath(finalURL, redirectTarget);
      if (welcomeRedirectPath) {
        window.location.href = welcomeRedirectPath;
        return;
      }

      if (redirectTarget) {
        window.location.href = redirectTarget;
        return;
      }

      try {
        const orgsResponse = await fetch("/organizations", {
          credentials: "include",
        });

        if (orgsResponse.ok) {
          const organizations = await orgsResponse.json();
          if (organizations.length === 1) {
            window.location.href = `/${organizations[0].id}`;
            return;
          }
        }
      } catch {
        // fall through to default redirect
      }

      window.location.href = finalURL;
    },
    [redirectTarget],
  );

  useEffect(() => {
    setLastUsedMethod(readLastUsedLoginMethod());
  }, []);

  useEffect(() => {
    if (!accountLoading && account) {
      window.location.href = safeRedirect || "/";
    }
  }, [account, accountLoading, safeRedirect]);

  useEffect(() => {
    setIsSignupMode(mode === "signup");
    setFormError(null);
    setMagicCodeStep("email");
    setMagicCode("");
    setShowPasswordLogin(false);
  }, [mode]);

  useEffect(() => {
    if (!magicLinkToken) return;

    const verifyMagicLink = async () => {
      setSubmitLoading(true);
      try {
        const { url, body } = buildMagicLinkVerifyRequest({
          token: magicLinkToken,
          inviteToken,
          redirectTarget,
          signupIntent: mode === "signup",
        });

        const response = await fetch(url, {
          method: "POST",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/x-www-form-urlencoded",
          },
          credentials: "include",
          body,
        });

        if (!response.ok) {
          setFormError("Invalid or expired link. Please request a new code.");
          setSubmitLoading(false);
          return;
        }

        const authRedirectURL = await getAuthRedirectURL(response);
        if (mode === "signup" && !getWelcomeRedirectPath(authRedirectURL, redirectTarget)) {
          clearPendingSignupAnalyticsPreference();
        }

        await handleRedirectAfterAuth(response, authRedirectURL);
      } catch {
        setFormError("Network error occurred");
        setSubmitLoading(false);
      }
    };

    verifyMagicLink();
  }, [magicLinkToken, inviteToken, redirectTarget, mode, handleRedirectAfterAuth]);

  useEffect(() => {
    let canceled = false;

    const fetchAuthConfig = async () => {
      try {
        const response = await fetch("/auth/config");
        if (!response.ok) {
          throw new Error("Failed to load auth configuration");
        }

        const data = (await response.json()) as AuthConfig;
        if (!canceled) {
          setAuthConfig({
            providers: data.providers || [],
            passwordLoginEnabled: Boolean(data.passwordLoginEnabled),
            signupEnabled: Boolean(data.signupEnabled),
            signupsBlockedByEnvironment: Boolean(data.signupsBlockedByEnvironment),
            magicCodeEnabled: Boolean(data.magicCodeEnabled),
          });
        }
      } catch {
        if (!canceled) {
          setConfigError("Failed to load login options.");
        }
      } finally {
        if (!canceled) {
          setConfigLoading(false);
        }
      }
    };

    fetchAuthConfig();

    return () => {
      canceled = true;
    };
  }, []);

  const providers = authConfig.providers || [];
  const allowedProviders = ["google", "github"];
  const activeProviders = allowedProviders.filter((provider) => providers.includes(provider));
  const hasProviders = activeProviders.length > 0;
  const canSignup = authConfig.signupEnabled || Boolean(inviteToken);

  useReportPageReady(!configLoading && !accountLoading, {
    failed: !!configError,
  });

  const canSignupWithPassword = authConfig.passwordLoginEnabled && canSignup;
  const canLoginWithPassword = authConfig.passwordLoginEnabled;
  const redirectQuery = safeRedirect ? `?redirect=${encodeURIComponent(safeRedirect)}` : "";
  const showProviderButtons = hasProviders && (!isSignupMode || canSignup);
  const canUseMagicCode = authConfig.magicCodeEnabled && (!isSignupMode || canSignup);
  const useMagicCodePrimary = canUseMagicCode && !showPasswordLogin;
  const showSignupUnavailable = !configLoading && isSignupMode && !canSignup;
  const hasConfiguredSignupWaitlist = hasSignupWaitlistConfig();
  const signupUnavailableReason = getSignupUnavailableReason(
    showSignupUnavailable,
    authConfig.signupsBlockedByEnvironment,
    hasConfiguredSignupWaitlist,
  );
  const showSignupWaitlist = signupUnavailableReason === "waitlist";
  const showSignupClosedNotice = signupUnavailableReason === "closed";
  const showSignupEntryPoint = canSignup || !authConfig.signupsBlockedByEnvironment;
  const showLastUsedMethodHints = !isSignupMode;
  const authErrorMessage = getAuthErrorMessage(authError, signupUnavailableReason);
  const showStandaloneProductUpdatesOptIn =
    isSignupMode && canSignup && magicCodeStep === "email" && !canSignupWithPassword && !useMagicCodePrimary;
  const visibleFormError = formError ?? authErrorMessage;

  const handleMagicCodeRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!magicCodeEmail.trim()) {
      setFormError("Email is required");
      return;
    }

    saveSignupPreference(isSignupMode, signupProductUpdatesOptIn, magicCodeEmail);

    setSubmitLoading(true);

    try {
      const response = await fetch("/auth/magic-code/request", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: buildMagicCodeRequestBody(magicCodeEmail, isSignupMode, redirectTarget),
      });

      if (!response.ok) {
        clearSignupPreference(isSignupMode);
        setFormError("Failed to send code. Please try again.");
        setSubmitLoading(false);
        return;
      }

      setMagicCodeStep("code");
      setSubmitLoading(false);
    } catch {
      clearSignupPreference(isSignupMode);
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  const handleMagicCodeVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!magicCode.trim()) {
      setFormError("Code is required");
      return;
    }

    setSubmitLoading(true);

    try {
      const url = redirectTarget
        ? `/auth/magic-code/verify?redirect=${encodeURIComponent(redirectTarget)}`
        : "/auth/magic-code/verify";

      const response = await fetch(url, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        credentials: "include",
        body: buildMagicCodeVerifyBody(magicCodeEmail, magicCode, isSignupMode, inviteToken),
      });

      if (!response.ok) {
        clearSignupPreference(isSignupMode);
        setFormError(await getMagicCodeVerifyError(response));
        setSubmitLoading(false);
        return;
      }

      const authRedirectURL = await getAuthRedirectURL(response);
      if (isSignupMode) {
        if (getWelcomeRedirectPath(authRedirectURL, redirectTarget)) {
          confirmSignupAnalyticsPreference({
            email: magicCodeEmail.trim(),
            productUpdatesOptIn: signupProductUpdatesOptIn,
          });
        } else {
          clearSignupPreference(true);
        }
      }

      await handleRedirectAfterAuth(response, authRedirectURL);
    } catch {
      clearSignupPreference(isSignupMode);
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  const handleMagicCodeBack = () => {
    setMagicCodeStep("email");
    setMagicCode("");
    setFormError(null);
  };

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!loginEmail.trim() || !loginPassword) {
      setFormError("Email and password are required");
      return;
    }

    setSubmitLoading(true);

    try {
      const formData = new URLSearchParams();
      formData.append("email", loginEmail.trim());
      formData.append("password", loginPassword);

      const url = redirectTarget ? `/login?redirect=${encodeURIComponent(redirectTarget)}` : "/login";
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        credentials: "include",
        body: formData.toString(),
      });

      if (!response.ok) {
        if (response.status === 401) {
          setFormError("Invalid email or password");
        } else {
          setFormError("Login failed. Please try again.");
        }
        setSubmitLoading(false);
        return;
      }

      recordLastUsedLoginMethod("password");
      await handleRedirectAfterAuth(response);
    } catch {
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  const handleSignupSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const validationError = getSignupValidationError({
      canSignup,
      firstName: signupFirstName,
      lastName: signupLastName,
      email: signupEmail,
      password: signupPassword,
      confirmPassword: signupConfirmPassword,
    });
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setSubmitLoading(true);

    try {
      const url = redirectTarget ? `/signup?redirect=${encodeURIComponent(redirectTarget)}` : "/signup";
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        credentials: "include",
        body: buildSignupBody(
          `${signupFirstName.trim()} ${signupLastName.trim()}`,
          signupEmail,
          signupPassword,
          inviteToken,
        ),
      });

      if (!response.ok) {
        setFormError(await getSignupError(response));
        setSubmitLoading(false);
        return;
      }

      confirmSignupAnalyticsPreference({
        email: signupEmail.trim(),
        productUpdatesOptIn: signupProductUpdatesOptIn,
      });

      await handleRedirectAfterAuth(response);
    } catch {
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  const handleProviderClick = (provider: string) => {
    saveSignupPreference(isSignupMode, signupProductUpdatesOptIn);

    recordLastUsedLoginMethod(provider as LastUsedLoginMethod);
  };

  const hasAnyFormMethod = canLoginWithPassword || canSignupWithPassword || showProviderButtons || useMagicCodePrimary;
  const providerButtonsNeedTopSpacing = useMagicCodePrimary && magicCodeStep === "email";

  const getHeading = () => {
    if (useMagicCodePrimary && magicCodeStep === "code") return "Check your email";
    if (showSignupClosedNotice) return "Signups are closed";
    if (showSignupWaitlist) return "SuperPlane Cloud";
    if (isSignupMode) return "Create your account";
    return "Welcome to SuperPlane";
  };

  const getSubheading = () => {
    if (showSignupClosedNotice) return "New accounts are not available.";
    if (showSignupWaitlist) return "Join the waitlist for access.";
    if (isSignupMode) return "Set up your account.";
    if (useMagicCodePrimary && magicCodeStep === "code") return `We sent a code to ${magicCodeEmail}`;
    return "Log in to continue";
  };

  return (
    <div className="min-h-screen bg-gray-400 flex items-center justify-center px-4 py-10">
      <div className="max-w-sm w-full rounded-3xl bg-white p-8 shadow-sm outline outline-gray-950/10 dark:bg-gray-900">
        <div className="text-center">
          <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto h-8 w-8" />
          <h1 className="mt-2 !text-lg font-medium text-gray-900">{getHeading()}</h1>
          <p className="mt-1 text-sm text-gray-600">{getSubheading()}</p>
        </div>

        <div className="pt-6">
          {configLoading && <p className="text-sm text-gray-500">Loading...</p>}

          {configError && (
            <div className="mb-4 rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-500">
              {configError}
            </div>
          )}

          {!configLoading && showSignupWaitlist && <SignupWaitlist />}
          {!configLoading && showSignupClosedNotice && <SignupClosedNotice />}

          {!configLoading && !isSignupMode && !hasAnyFormMethod && (
            <p className="text-sm text-gray-500">No login methods are configured.</p>
          )}

          {!configLoading && visibleFormError && (
            <div className="mb-4 rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-500">
              {visibleFormError}
            </div>
          )}

          {!configLoading && useMagicCodePrimary && magicCodeStep === "email" && (
            <form onSubmit={handleMagicCodeRequest} className="space-y-4">
              <div className="space-y-2">
                <Label>Email</Label>
                <Input
                  type="email"
                  name="email"
                  placeholder="you@example.com"
                  required
                  autoComplete="email"
                  value={magicCodeEmail}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMagicCodeEmail(e.target.value)}
                />
              </div>

              {isSignupMode && (
                <ProductUpdatesOptIn
                  checked={signupProductUpdatesOptIn}
                  onCheckedChange={setSignupProductUpdatesOptIn}
                />
              )}

              <LoadingButton type="submit" loading={submitLoading} loadingText="Sending code..." className="w-full">
                Continue with email
              </LoadingButton>
            </form>
          )}

          {!configLoading && useMagicCodePrimary && magicCodeStep === "code" && (
            <form onSubmit={handleMagicCodeVerify} className="space-y-4">
              <div className="space-y-2">
                <Label>Code</Label>
                <Input
                  type="text"
                  name="code"
                  placeholder="Enter 6-digit code"
                  required
                  autoComplete="one-time-code"
                  inputMode="numeric"
                  maxLength={7}
                  value={magicCode}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMagicCode(e.target.value)}
                />
              </div>

              <LoadingButton type="submit" loading={submitLoading} loadingText="Verifying..." className="w-full">
                {isSignupMode ? "Create account" : "Sign in"}
              </LoadingButton>

              <div className="text-center">
                <button
                  type="button"
                  onClick={handleMagicCodeBack}
                  className="text-sm text-gray-500 underline underline-offset-2"
                >
                  Use a different email
                </button>
              </div>
            </form>
          )}

          {!configLoading && !isSignupMode && !useMagicCodePrimary && canLoginWithPassword && (
            <form onSubmit={handleLoginSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label>Email</Label>
                <Input
                  type="email"
                  name="email"
                  placeholder="Email"
                  required
                  autoComplete="email"
                  value={loginEmail}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLoginEmail(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label>Password</Label>
                <Input
                  type="password"
                  name="password"
                  placeholder="Password"
                  required
                  autoComplete="current-password"
                  value={loginPassword}
                  className="ph-no-capture"
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLoginPassword(e.target.value)}
                />
              </div>

              <div>
                <LoadingButton type="submit" loading={submitLoading} loadingText="Logging in..." className="w-full">
                  Login
                </LoadingButton>
                {showLastUsedMethodHints && lastUsedMethod === "password" && <LastUsedHint label="email" />}
              </div>
            </form>
          )}

          {!configLoading && isSignupMode && canSignupWithPassword && (
            <form onSubmit={handleSignupSubmit} className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>First name</Label>
                  <Input
                    type="text"
                    name="firstName"
                    placeholder="First name"
                    required
                    autoComplete="given-name"
                    value={signupFirstName}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSignupFirstName(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Last name</Label>
                  <Input
                    type="text"
                    name="lastName"
                    placeholder="Last name"
                    required
                    autoComplete="family-name"
                    value={signupLastName}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSignupLastName(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>Email</Label>
                <Input
                  type="email"
                  name="email"
                  placeholder="Email"
                  required
                  autoComplete="email"
                  value={signupEmail}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSignupEmail(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label>Password</Label>
                <Input
                  type="password"
                  name="password"
                  placeholder="Password"
                  required
                  autoComplete="new-password"
                  value={signupPassword}
                  className="ph-no-capture"
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSignupPassword(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label>Repeat password</Label>
                <Input
                  type="password"
                  name="passwordConfirm"
                  placeholder="Repeat password"
                  required
                  autoComplete="new-password"
                  value={signupConfirmPassword}
                  className="ph-no-capture"
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSignupConfirmPassword(e.target.value)}
                />
              </div>

              <ProductUpdatesOptIn checked={signupProductUpdatesOptIn} onCheckedChange={setSignupProductUpdatesOptIn} />

              <LoadingButton
                type="submit"
                disabled={!canSignup}
                loading={submitLoading}
                loadingText="Creating account..."
                className="w-full"
              >
                Create account
              </LoadingButton>
            </form>
          )}

          {!configLoading &&
            useMagicCodePrimary &&
            !isSignupMode &&
            magicCodeStep === "email" &&
            canLoginWithPassword && (
              <div className="mt-4 text-center">
                <button
                  type="button"
                  onClick={() => {
                    setShowPasswordLogin(true);
                    setFormError(null);
                  }}
                  className="text-sm text-gray-500 underline underline-offset-2"
                >
                  Sign in with password instead
                </button>
              </div>
            )}

          {!configLoading &&
            !isSignupMode &&
            showPasswordLogin &&
            !useMagicCodePrimary &&
            authConfig.magicCodeEnabled && (
              <div className="mt-4 text-center">
                <button
                  type="button"
                  onClick={() => {
                    setShowPasswordLogin(false);
                    setFormError(null);
                  }}
                  className="text-sm text-gray-500 underline underline-offset-2"
                >
                  Sign in with email code instead
                </button>
              </div>
            )}

          {!configLoading &&
            showProviderButtons &&
            (isSignupMode
              ? canSignupWithPassword
              : useMagicCodePrimary
                ? magicCodeStep === "email"
                : canLoginWithPassword) && (
              <div className="my-5 flex items-center gap-3 text-sm text-gray-800">
                <div className="h-px flex-1 bg-gray-300" />
                <span>or</span>
                <div className="h-px flex-1 bg-gray-300" />
              </div>
            )}

          {!configLoading && showStandaloneProductUpdatesOptIn && (
            <div className="mb-4">
              <ProductUpdatesOptIn checked={signupProductUpdatesOptIn} onCheckedChange={setSignupProductUpdatesOptIn} />
            </div>
          )}

          {!configLoading && showProviderButtons && (!useMagicCodePrimary || magicCodeStep === "email") && (
            <div className={providerButtonsNeedTopSpacing ? "mt-4 space-y-3" : "space-y-3"}>
              {activeProviders.map((provider) => (
                <div key={provider}>
                  <Button variant="outline" className="w-full justify-center gap-2" asChild>
                    <a
                      href={providerAuthPath(provider, redirectQuery, isSignupMode)}
                      onClick={() => handleProviderClick(provider)}
                    >
                      {provider === "github" && (
                        <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
                          <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                        </svg>
                      )}
                      {provider === "google" && (
                        <svg className="h-4 w-4" viewBox="0 0 24 24" aria-hidden>
                          <path
                            fill="#4285F4"
                            d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                          />
                          <path
                            fill="#34A853"
                            d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                          />
                          <path
                            fill="#FBBC05"
                            d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                          />
                          <path
                            fill="#EA4335"
                            d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                          />
                        </svg>
                      )}
                      <span>Continue with {getProviderLabel(provider)}</span>
                    </a>
                  </Button>
                  {showLastUsedMethodHints && lastUsedMethod === provider && (
                    <LastUsedHint label={getProviderLabel(provider)} />
                  )}
                </div>
              ))}
            </div>
          )}

          {!configLoading && !isSignupMode && showSignupEntryPoint && !useMagicCodePrimary && (
            <div className="mt-6 text-sm text-gray-500">
              {"Don't have an account? "}
              <Link to={`/signup${redirectQuery}`} className="font-medium text-gray-900 underline underline-offset-2">
                Create an account
              </Link>
            </div>
          )}

          {!configLoading && isSignupMode && (
            <div className="mt-6 text-sm text-gray-500">
              Already have an account?{" "}
              <Link to={`/login${redirectQuery}`} className="font-medium text-gray-900 underline underline-offset-2">
                Sign in
              </Link>
            </div>
          )}

          {!configLoading &&
            useMagicCodePrimary &&
            magicCodeStep === "email" &&
            !isSignupMode &&
            showSignupEntryPoint && (
              <div className="mt-6 text-center text-sm text-gray-500">
                {"Don't have an account? "}
                <Link to={`/signup${redirectQuery}`} className="font-medium text-gray-900 underline underline-offset-2">
                  Sign up
                </Link>
              </div>
            )}
        </div>
      </div>
    </div>
  );
};
