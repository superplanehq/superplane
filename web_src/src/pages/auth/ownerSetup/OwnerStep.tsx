import React from "react";
import superplaneLogo from "@/assets/superplane.svg";
import { Input, InputGroup } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
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
  onNext: (event: React.FormEvent) => void;
};

const OwnerStepHeader = (
  <div className="mb-8 text-center">
    <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto mb-4 h-8 w-8 dark:brightness-0 dark:invert" />
    <h4 className="mb-1 text-xl font-medium text-gray-800 dark:text-white">Set up owner account</h4>
    <Text className="text-gray-800 dark:text-gray-300">Create an account for this SuperPlane instance.</Text>
  </div>
);

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
  onNext,
}) => (
  <>
    {OwnerStepHeader}
    <form onSubmit={onNext} className="space-y-4">
      <ErrorBanner message={error} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Label className="mb-2 block text-left">
            First Name <span className="text-gray-800 dark:text-gray-300">*</span>
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
          <Label className="mb-2 block text-left">
            Last Name <span className="text-gray-800 dark:text-gray-300">*</span>
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
        <Label className="mb-2 block text-left">
          Email <span className="text-gray-800 dark:text-gray-300">*</span>
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
        <Label className="mb-2 block text-left">
          Password <span className="text-gray-800 dark:text-gray-300">*</span>
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
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            8+ characters, at least 1 number and 1 capital letter
          </p>
        )}
      </div>
      <div>
        <Label className="mb-2 block text-left">
          Confirm Password <span className="text-gray-800 dark:text-gray-300">*</span>
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
      <div className="flex justify-end">
        <Button type="submit" disabled={loading}>
          {loading ? "Saving..." : "Next"}
        </Button>
      </div>
    </form>
  </>
);
