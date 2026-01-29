/**
 * Authentication service for non-API auth endpoints
 * Handles login, signup, and account-related HTTP requests
 *
 * Note: For /api/v1/* endpoints, use the generated SDK in @/api-client
 */

export interface AuthConfig {
  providers: string[];
  passwordLoginEnabled: boolean;
  signupEnabled: boolean;
}

export interface Account {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
}

export interface LoginParams {
  email: string;
  password: string;
  redirect?: string;
}

export interface SignupParams {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
  inviteToken?: string;
  redirect?: string;
}

export interface AccountResponse {
  status: number;
  account?: Account;
  setupRequired?: boolean;
}

/**
 * Fetch authentication configuration
 */
export async function fetchAuthConfig(): Promise<AuthConfig> {
  const response = await fetch("/auth/config", {
    method: "GET",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load auth configuration");
  }

  return response.json();
}

/**
 * Login with email and password
 */
export async function login(params: LoginParams): Promise<Response> {
  const formData = new URLSearchParams();
  formData.append("email", params.email.trim());
  formData.append("password", params.password);

  const url = params.redirect
    ? `/login?redirect=${encodeURIComponent(params.redirect)}`
    : "/login";

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    credentials: "include",
    body: formData.toString(),
  });

  return response;
}

/**
 * Sign up with user details
 */
export async function signup(params: SignupParams): Promise<Response> {
  const formData = new URLSearchParams();
  formData.append("name", `${params.firstName.trim()} ${params.lastName.trim()}`);
  formData.append("email", params.email.trim());
  formData.append("password", params.password);

  if (params.inviteToken) {
    formData.append("invite_token", params.inviteToken);
  }

  const url = params.redirect
    ? `/signup?redirect=${encodeURIComponent(params.redirect)}`
    : "/signup";

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    credentials: "include",
    body: formData.toString(),
  });

  return response;
}

/**
 * Fetch current user's account information
 */
export async function fetchAccount(): Promise<AccountResponse> {
  const response = await fetch("/account", {
    method: "GET",
    credentials: "include",
    redirect: "manual",
  });

  if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
    return {
      status: response.status,
      setupRequired: true,
    };
  }

  if (response.status === 200) {
    const account = await response.json();
    return {
      status: response.status,
      account,
    };
  }

  return {
    status: response.status,
  };
}
