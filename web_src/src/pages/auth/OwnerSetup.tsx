import React, { useState } from "react";
import { Input, InputGroup } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { Button } from "../../ui/button";
import superplaneLogo from "../../assets/superplane.svg";

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromName, setSmtpFromName] = useState("");
  const [smtpFromEmail, setSmtpFromEmail] = useState("");
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);
  const [step, setStep] = useState<"owner" | "smtpPrompt" | "smtpConfig">("owner");
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

  const validateOwnerFields = () => {
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

  const validateSMTPFields = () => {
    const errors: Record<string, string> = {};

    if (!smtpHost.trim()) {
      errors.smtpHost = "SMTP host is required.";
    }

    if (!smtpPort.trim()) {
      errors.smtpPort = "SMTP port is required.";
    } else if (!/^[0-9]+$/.test(smtpPort.trim())) {
      errors.smtpPort = "SMTP port must be a number.";
    }

    if (!smtpFromEmail.trim()) {
      errors.smtpFromEmail = "SMTP from email is required.";
    } else if (!isEmailValid(smtpFromEmail.trim())) {
      errors.smtpFromEmail = "Please enter a valid from email address.";
    }

    if (smtpUsername.trim() && !smtpPassword) {
      errors.smtpPassword = "SMTP password is required when username is provided.";
    }

    setFieldErrors(errors);
    return errors;
  };

  const submitSetup = async (enableSMTP: boolean) => {
    setError(null);

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
          smtp_enabled: enableSMTP,
          smtp_host: enableSMTP ? smtpHost.trim() : "",
          smtp_port: enableSMTP && smtpPort ? Number(smtpPort) : 0,
          smtp_username: enableSMTP ? smtpUsername.trim() : "",
          smtp_password: enableSMTP ? smtpPassword : "",
          smtp_from_name: enableSMTP ? smtpFromName.trim() : "",
          smtp_from_email: enableSMTP ? smtpFromEmail.trim() : "",
          smtp_use_tls: enableSMTP ? smtpUseTLS : false,
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

  const handleOwnerNext = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateOwnerFields();
    if (Object.keys(errors).length > 0) {
      return;
    }
    setStep("smtpPrompt");
  };

  const handleSkipSMTP = () => {
    submitSetup(false);
  };

  const handleEnableSMTP = () => {
    setStep("smtpConfig");
  };

  const handleSubmitSMTP = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateSMTPFields();
    if (Object.keys(errors).length > 0) {
      return;
    }
    submitSetup(true);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800 px-4">
      <div className="max-w-lg w-full bg-white dark:bg-zinc-900 rounded-lg shadow-xl p-8">
        {step === "owner" && (
          <div className="text-center mb-8">
            <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto mb-4 h-8 w-8" />
            <h4 className="text-2xl font-bold text-gray-800 dark:text-white mb-2">Set up owner account</h4>
            <Text className="text-gray-500 dark:text-gray-400">Create an account for this SuperPlane instance.</Text>
          </div>
        )}

        {step === "owner" && (
          <form onSubmit={handleOwnerNext} className="space-y-6">
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
        )}

        {step === "smtpPrompt" && (
          <div className="space-y-6">
            {error && (
              <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
              </div>
            )}

            <div className="text-left">
              <h4 className="text-lg font-semibold text-gray-800 dark:text-white">Set up email delivery?</h4>
              <Text className="text-gray-500 dark:text-gray-400">
                Configure SMTP now to receive notifications. You can skip and set it up later.
              </Text>
            </div>

            <div className="flex gap-3">
              <Button type="button" className="w-full" disabled={loading} onClick={handleEnableSMTP}>
                Set up SMTP
              </Button>
              <Button type="button" className="w-full" variant="secondary" disabled={loading} onClick={handleSkipSMTP}>
                Do this later
              </Button>
            </div>
          </div>
        )}

        {step === "smtpConfig" && (
          <form onSubmit={handleSubmitSMTP} className="space-y-6">
            {error && (
              <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
              </div>
            )}

            <div className="text-left">
              <h4 className="text-lg font-semibold text-gray-800 dark:text-white">SMTP configuration</h4>
              <Text className="text-gray-500 dark:text-gray-400">Configure email delivery for this instance.</Text>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                SMTP Host <span className="text-red-500">*</span>
              </label>
              <InputGroup>
                <Input
                  type="text"
                  value={smtpHost}
                  onChange={(e) => setSmtpHost(e.target.value)}
                  placeholder="smtp.example.com"
                  className={fieldErrors.smtpHost ? "border-red-500" : ""}
                />
              </InputGroup>
              {fieldErrors.smtpHost && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpHost}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                SMTP Port <span className="text-red-500">*</span>
              </label>
              <InputGroup>
                <Input
                  type="text"
                  value={smtpPort}
                  onChange={(e) => setSmtpPort(e.target.value)}
                  placeholder="587"
                  className={fieldErrors.smtpPort ? "border-red-500" : ""}
                />
              </InputGroup>
              {fieldErrors.smtpPort && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpPort}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                SMTP Username
              </label>
              <InputGroup>
                <Input
                  type="text"
                  value={smtpUsername}
                  onChange={(e) => setSmtpUsername(e.target.value)}
                  placeholder="smtp-user"
                />
              </InputGroup>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                SMTP Password
              </label>
              <InputGroup>
                <Input
                  type="password"
                  value={smtpPassword}
                  onChange={(e) => setSmtpPassword(e.target.value)}
                  placeholder="SMTP password"
                  className={fieldErrors.smtpPassword ? "border-red-500" : ""}
                />
              </InputGroup>
              {fieldErrors.smtpPassword && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpPassword}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                From Name
              </label>
              <InputGroup>
                <Input
                  type="text"
                  value={smtpFromName}
                  onChange={(e) => setSmtpFromName(e.target.value)}
                  placeholder="SuperPlane"
                />
              </InputGroup>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
                From Email <span className="text-red-500">*</span>
              </label>
              <InputGroup>
                <Input
                  type="email"
                  value={smtpFromEmail}
                  onChange={(e) => setSmtpFromEmail(e.target.value)}
                  placeholder="noreply@example.com"
                  className={fieldErrors.smtpFromEmail ? "border-red-500" : ""}
                />
              </InputGroup>
              {fieldErrors.smtpFromEmail && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpFromEmail}</p>
              )}
            </div>

            <label className="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input type="checkbox" checked={smtpUseTLS} onChange={(e) => setSmtpUseTLS(e.target.checked)} />
              Use TLS (STARTTLS)
            </label>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Saving..." : "Finish setup"}
            </Button>
          </form>
        )}
      </div>
    </div>
  );
};

export default OwnerSetup;
