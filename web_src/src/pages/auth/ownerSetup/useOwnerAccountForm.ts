import { useState } from "react";

function isEmailValid(email: string) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

function isPasswordValid(password: string) {
  if (password.length < 8) return false;
  if (!/[0-9]/.test(password)) return false;
  if (!/[A-Z]/.test(password)) return false;
  return true;
}

function validateOwnerFields(values: {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  confirmPassword: string;
}): Record<string, string> {
  const errors: Record<string, string> = {};

  if (!values.email.trim()) {
    errors.email = "Email is required.";
  } else if (!isEmailValid(values.email.trim())) {
    errors.email = "Please enter a valid email address.";
  }

  if (!values.firstName.trim()) {
    errors.firstName = "First name is required.";
  }

  if (!values.lastName.trim()) {
    errors.lastName = "Last name is required.";
  }

  if (!values.password) {
    errors.password = "Password is required.";
  } else if (!isPasswordValid(values.password)) {
    errors.password = "Password must be 8+ characters with at least 1 number and 1 capital letter.";
  }

  if (!values.confirmPassword) {
    errors.confirmPassword = "Please confirm your password.";
  } else if (values.confirmPassword !== values.password) {
    errors.confirmPassword = "Passwords do not match.";
  }

  return errors;
}

async function submitOwnerSetup(args: {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  setError: (error: string | null) => void;
  setLoading: (loading: boolean) => void;
  setPendingOrganizationSlug: (slug: string | null) => void;
  onOwnerCreated: (step: "license") => void;
}) {
  args.setError(null);
  args.setLoading(true);

  try {
    const response = await fetch("/api/v1/setup-owner", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        email: args.email.trim(),
        first_name: args.firstName.trim(),
        last_name: args.lastName.trim(),
        password: args.password,
      }),
    });

    if (!response.ok) {
      try {
        const data = await response.json();
        args.setError(data.message || "Failed to set up owner account");
      } catch {
        if (response.status === 409) {
          args.setError("This instance is already initialized.");
        } else {
          args.setError(`Failed to set up owner account (${response.status})`);
        }
      }
      return;
    }

    const data: { organization_id: string; organization_slug: string } = await response.json();
    args.setPendingOrganizationSlug(data.organization_slug);
    args.onOwnerCreated("license");
  } catch {
    args.setError("Network error occurred");
  } finally {
    args.setLoading(false);
  }
}

export function useOwnerAccountForm(actions: {
  setPendingOrganizationSlug: (slug: string | null) => void;
  onOwnerCreated: (step: "license") => void;
}) {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const handleOwnerSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const errors = validateOwnerFields({ email, firstName, lastName, password, confirmPassword });
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }
    void submitOwnerSetup({
      email,
      firstName,
      lastName,
      password,
      setError,
      setLoading,
      setPendingOrganizationSlug: actions.setPendingOrganizationSlug,
      onOwnerCreated: actions.onOwnerCreated,
    });
  };

  return {
    email,
    setEmail,
    firstName,
    setFirstName,
    lastName,
    setLastName,
    password,
    setPassword,
    confirmPassword,
    setConfirmPassword,
    loading,
    error,
    fieldErrors,
    handleOwnerSubmit,
  };
}
