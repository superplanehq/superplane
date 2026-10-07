import React from "react";
import { Input, InputGroup } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { ErrorBanner } from "./ErrorBanner";

type OwnerStepProps = {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  confirmPassword: string;
  loading: boolean;
  error: string | null;
  fieldErrors: Record<string, string>;
  onEmailChange: (value: string) => void;
  onFirstNameChange: (value: string) => void;
  onLastNameChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onConfirmPasswordChange: (value: string) => void;
  onSubmit: (event: React.FormEvent) => void;
};

const fieldLabelClass = "mb-2 block text-left text-[13px] font-medium";

export const OwnerStep: React.FC<OwnerStepProps> = ({
  email,
  firstName,
  lastName,
  password,
  confirmPassword,
  loading,
  error,
  fieldErrors,
  onEmailChange,
  onFirstNameChange,
  onLastNameChange,
  onPasswordChange,
  onConfirmPasswordChange,
  onSubmit,
}) => (
  <>
    <FirstRunHeading headline="Create the owner account" size="display">
      <p className="text-[15px] leading-6 text-muted-foreground">This account administers the installation.</p>
    </FirstRunHeading>
    <form onSubmit={onSubmit} className="mt-8 space-y-4 text-left">
      <ErrorBanner message={error} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Label className={fieldLabelClass}>
            First name <span className="text-foreground">*</span>
          </Label>
          <InputGroup>
            <Input
              type="text"
              value={firstName}
              onChange={(event) => onFirstNameChange(event.target.value)}
              placeholder="First name"
              className={fieldErrors.firstName ? "border-red-500" : ""}
            />
          </InputGroup>
          {fieldErrors.firstName && (
            <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.firstName}</p>
          )}
        </div>
        <div>
          <Label className={fieldLabelClass}>
            Last name <span className="text-foreground">*</span>
          </Label>
          <InputGroup>
            <Input
              type="text"
              value={lastName}
              onChange={(event) => onLastNameChange(event.target.value)}
              placeholder="Last name"
              className={fieldErrors.lastName ? "border-red-500" : ""}
            />
          </InputGroup>
          {fieldErrors.lastName && (
            <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.lastName}</p>
          )}
        </div>
      </div>
      <div>
        <Label className={fieldLabelClass}>
          Email <span className="text-foreground">*</span>
        </Label>
        <InputGroup>
          <Input
            type="email"
            value={email}
            onChange={(event) => onEmailChange(event.target.value)}
            placeholder="you@example.com"
            className={fieldErrors.email ? "border-red-500" : ""}
          />
        </InputGroup>
        {fieldErrors.email && <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.email}</p>}
      </div>
      <div>
        <Label className={fieldLabelClass}>
          Password <span className="text-foreground">*</span>
        </Label>
        <InputGroup>
          <Input
            type="password"
            value={password}
            onChange={(event) => onPasswordChange(event.target.value)}
            placeholder="Password"
            className={fieldErrors.password ? "border-red-500 ph-no-capture" : "ph-no-capture"}
          />
        </InputGroup>
        {fieldErrors.password ? (
          <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.password}</p>
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">8+ characters, at least 1 number and 1 capital letter</p>
        )}
      </div>
      <div>
        <Label className={fieldLabelClass}>
          Confirm password <span className="text-foreground">*</span>
        </Label>
        <InputGroup>
          <Input
            type="password"
            value={confirmPassword}
            onChange={(event) => onConfirmPasswordChange(event.target.value)}
            placeholder="Confirm password"
            className={fieldErrors.confirmPassword ? "border-red-500 ph-no-capture" : "ph-no-capture"}
          />
        </InputGroup>
        {fieldErrors.confirmPassword && (
          <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.confirmPassword}</p>
        )}
      </div>
      <Button type="submit" className="mt-4 min-w-40" disabled={loading}>
        {loading ? "Saving..." : "Continue"}
      </Button>
    </form>
  </>
);
