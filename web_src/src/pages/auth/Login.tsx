import React, { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import superplaneLogo from "@/assets/superplane.svg";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAccount } from "../../contexts/AccountContext";

type AuthConfig = {
  providers: string[];
  passwordLoginEnabled: boolean;
  signupEnabled: boolean;
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

export const Login: React.FC = () => {
  const [authConfig, setAuthConfig] = useState<AuthConfig>({
    providers: [],
    passwordLoginEnabled: false,
    signupEnabled: false,
  });
  const [configLoading, setConfigLoading] = useState(true);
  const [configError, setConfigError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [isSignupMode, setIsSignupMode] = useState(false);
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [signupFirstName, setSignupFirstName] = useState("");
  const [signupLastName, setSignupLastName] = useState("");
  const [signupEmail, setSignupEmail] = useState("");
  const [signupPassword, setSignupPassword] = useState("");
  const [signupConfirmPassword, setSignupConfirmPassword] = useState("");
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

  useEffect(() => {
    if (!accountLoading && account) {
      window.location.href = safeRedirect || "/";
    }
  }, [account, accountLoading, safeRedirect]);

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
          });
        }
      } catch (err) {
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
  const canSignup = authConfig.signupEnabled || inviteToken;
  const canSignupWithPassword = authConfig.passwordLoginEnabled && canSignup;
  const canLoginWithPassword = authConfig.passwordLoginEnabled;
  const redirectQuery = safeRedirect ? `?redirect=${encodeURIComponent(safeRedirect)}` : "";
  const redirectTarget = safeRedirect || "";
  const showProviderButtons = hasProviders && (!isSignupMode || canSignup);

  useEffect(() => {
    if (!canSignup && isSignupMode) {
      setIsSignupMode(false);
      setFormError(null);
    }
  }, [canSignup, isSignupMode]);

  const handleToggleMode = (nextMode: "login" | "signup") => {
    setIsSignupMode(nextMode === "signup");
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

      const finalURL = response.url || "/";
      window.location.href = finalURL;
    } catch {
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  const handleSignupSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!canSignup) {
      setFormError("Signups are currently disabled.");
      return;
    }

    if (!signupFirstName.trim() || !signupLastName.trim()) {
      setFormError("First and last names are required");
      return;
    }

    if (!signupEmail.trim() || !signupPassword || !signupConfirmPassword) {
      setFormError("Email and password are required");
      return;
    }

    if (signupPassword !== signupConfirmPassword) {
      setFormError("Passwords do not match");
      return;
    }

    setSubmitLoading(true);

    try {
      const formData = new URLSearchParams();
      formData.append("name", `${signupFirstName.trim()} ${signupLastName.trim()}`);
      formData.append("email", signupEmail.trim());
      formData.append("password", signupPassword);
      if (inviteToken) {
        formData.append("invite_token", inviteToken);
      }

      const url = redirectTarget ? `/signup?redirect=${encodeURIComponent(redirectTarget)}` : "/signup";
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        credentials: "include",
        body: formData.toString(),
      });

      if (!response.ok) {
        if (response.status === 409) {
          setFormError("Account already exists. Please sign in.");
        } else {
          const errorText = await response.text();
          setFormError(errorText || "Signup failed. Please try again.");
        }
        setSubmitLoading(false);
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

      const finalURL = response.url || "/";
      window.location.href = finalURL;
    } catch {
      setFormError("Network error occurred");
      setSubmitLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-100 flex items-center justify-center px-4 py-10">
      <div className="max-w-sm w-full bg-white dark:bg-gray-900 rounded-lg outline outline-gray-950/10 shadow-sm p-8">
        <div className="text-center">
          <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto h-8 w-8" />
          <h1 className="mt-4 !text-lg font-medium text-gray-900">
            {isSignupMode ? "Create your account" : "Welcome to SuperPlane"}
          </h1>
          <p className="mt-1 text-sm text-gray-800">{isSignupMode ? "Set up your account." : "Log in to continue."}</p>
        </div>

        <div className="pt-8">
          {configLoading && <p className="text-sm text-gray-500">Loading...</p>}

          {configError && (
            <div className="mb-4 rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-500">
              {configError}
            </div>
          )}

          {!configLoading && isSignupMode && !canSignup && (
            <p className="text-sm text-gray-500">Signups are currently disabled.</p>
          )}

          {!configLoading &&
            !isSignupMode &&
            !showProviderButtons &&
            !canLoginWithPassword &&
            !canSignupWithPassword && <p className="text-sm text-gray-500">No login methods are configured.</p>}

          {!configLoading && formError && (
            <div className="mb-4 rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-500">
              {formError}
            </div>
          )}

          {!configLoading && !isSignupMode && canLoginWithPassword && (
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
                  onChange={(e) => setLoginEmail(e.target.value)}
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
                  onChange={(e) => setLoginPassword(e.target.value)}
                />
              </div>

              <Button type="submit" disabled={submitLoading} className="w-full">
                {submitLoading ? "Logging in..." : "Login"}
              </Button>
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
                    onChange={(e) => setSignupFirstName(e.target.value)}
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
                    onChange={(e) => setSignupLastName(e.target.value)}
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
                  onChange={(e) => setSignupEmail(e.target.value)}
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
                  onChange={(e) => setSignupPassword(e.target.value)}
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
                  onChange={(e) => setSignupConfirmPassword(e.target.value)}
                />
              </div>

              <Button type="submit" disabled={submitLoading || !canSignup} className="w-full">
                {submitLoading ? "Creating account..." : "Create account"}
              </Button>
            </form>
          )}

          {!configLoading && showProviderButtons && (isSignupMode ? canSignupWithPassword : canLoginWithPassword) && (
            <div className="my-5 flex items-center gap-3 text-sm text-gray-800">
              <div className="h-px flex-1 bg-gray-300" />
              <span>or</span>
              <div className="h-px flex-1 bg-gray-300" />
            </div>
          )}

          {!configLoading && showProviderButtons && (
            <div className="space-y-3">
              {activeProviders.map((provider) => (
                <Button key={provider} variant="outline" className="w-full justify-center gap-2" asChild>
                  <a href={`/auth/${provider}${redirectQuery}`}>
                    {provider === "github" && (
                      <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
                        <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                      </svg>
                    )}
                    {provider === "google" && (
                      <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
                        <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                        <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                        <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                        <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                      </svg>
                    )}
                    <span>Continue with {getProviderLabel(provider)}</span>
                  </a>
                </Button>
              ))}
            </div>
          )}

          {!configLoading && !isSignupMode && canSignup && (
            <div className="mt-6 text-sm text-gray-500">
              {"Don't have an account? "}
              <button
                type="button"
                onClick={() => handleToggleMode("signup")}
                className="font-medium text-gray-900 underline underline-offset-2"
              >
                Create an account
              </button>
            </div>
          )}

          {!configLoading && isSignupMode && (
            <div className="mt-6 text-sm text-gray-500">
              Already have an account?{" "}
              <button
                type="button"
                onClick={() => handleToggleMode("login")}
                className="font-medium text-gray-900 underline underline-offset-2"
              >
                Sign in
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
