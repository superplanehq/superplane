import React, { useState } from "react";
import { Input, InputGroup } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { Button } from "../../ui/button";

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const isEmailValid = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  const isPasswordValid = (password: string) => {
    if (password.length < 8) return false;
    if (!/[0-9]/.test(password)) return false;
    if (!/[A-Z]/.test(password)) return false;
    return true;
  };

  const validateAllFields = () => {
    const errors: Record<string, string> = {};

    if (!email.trim()) {
      errors.email = "Email is required.";
    } else if (!isEmailValid(email.trim())) {
      errors.email = "Please enter a valid email address.";
    }

    if (!firstName.trim()) {
      errors.firstName = "First name is required.";
    }

    if (!lastName.trim()) {
      errors.lastName = "Last name is required.";
    }

    if (!password) {
      errors.password = "Password is required.";
    } else if (!isPasswordValid(password)) {
      errors.password = "Password must be 8+ characters with at least 1 number and 1 capital letter.";
    }

    if (!confirmPassword) {
      errors.confirmPassword = "Please confirm your password.";
    } else if (confirmPassword !== password) {
      errors.confirmPassword = "Passwords do not match.";
    }

    setFieldErrors(errors);
    return errors;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validate all fields synchronously
    const errors = validateAllFields();

    // Check if there are any errors
    if (Object.keys(errors).length > 0) {
      return;
    }

    setLoading(true);

    try {
      const response = await fetch("/api/v1/setup-owner", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          email: email.trim(),
          first_name: firstName.trim(),
          last_name: lastName.trim(),
          password,
        }),
      });

      if (!response.ok) {
        try {
          const data = await response.json();
          setError(data.message || "Failed to set up owner account");
        } catch {
          if (response.status === 409) {
            setError("This instance is already initialized.");
          } else {
            setError(`Failed to set up owner account (${response.status})`);
          }
        }
        return;
      }

      const data: { organization_id: string } = await response.json();
      window.location.href = `/${data.organization_id}`;
    } catch (err) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800 px-4">
      <div className="max-w-lg w-full bg-white dark:bg-zinc-900 rounded-lg shadow-xl p-8">
        <div className="text-center mb-8">
          <h4 className="text-2xl font-bold text-gray-800 dark:text-white mb-2">Set up owner account</h4>
          <Text className="text-gray-600 dark:text-gray-400">Create an account for this SuperPlane instance.</Text>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              Email <span className="text-red-500">*</span>
            </label>
            <InputGroup>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className={fieldErrors.email ? "border-red-500" : ""}
              />
            </InputGroup>
            {fieldErrors.email && <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.email}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              First Name <span className="text-red-500">*</span>
            </label>
            <InputGroup>
              <Input
                type="text"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="First name"
                className={fieldErrors.firstName ? "border-red-500" : ""}
              />
            </InputGroup>
            {fieldErrors.firstName && (
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.firstName}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              Last Name <span className="text-red-500">*</span>
            </label>
            <InputGroup>
              <Input
                type="text"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="Last name"
                className={fieldErrors.lastName ? "border-red-500" : ""}
              />
            </InputGroup>
            {fieldErrors.lastName && (
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.lastName}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              Password <span className="text-red-500">*</span>
            </label>
            <InputGroup>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Password"
                className={fieldErrors.password ? "border-red-500" : ""}
              />
            </InputGroup>
            {fieldErrors.password ? (
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.password}</p>
            ) : (
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                8+ characters, at least 1 number and 1 capital letter
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              Confirm Password <span className="text-red-500">*</span>
            </label>
            <InputGroup>
              <Input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm password"
                className={fieldErrors.confirmPassword ? "border-red-500" : ""}
              />
            </InputGroup>
            {fieldErrors.confirmPassword && (
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.confirmPassword}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "Saving..." : "Next"}
          </Button>
        </form>
      </div>
    </div>
  );
};

export default OwnerSetup;
