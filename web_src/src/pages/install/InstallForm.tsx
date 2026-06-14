import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { MAX_APP_NAME_LENGTH } from "./constants";
import { InstallParamsForm } from "./InstallParamsForm";
import type { InstallParam, OrganizationOption } from "./types";

interface InstallFormProps {
  name: string;
  nameError: string;
  organizationId: string;
  organizations: OrganizationOption[];
  showOrganizationPicker: boolean;
  isInstalling: boolean;
  installParams?: InstallParam[];
  installParamValues: Record<string, string>;
  onNameChange: (value: string) => void;
  onOrganizationChange: (value: string) => void;
  onInstallParamChange: (name: string, value: string) => void;
  onSubmit: () => void;
}

export function InstallForm({
  name,
  nameError,
  organizationId,
  organizations,
  showOrganizationPicker,
  isInstalling,
  installParams,
  installParamValues,
  onNameChange,
  onOrganizationChange,
  onInstallParamChange,
  onSubmit,
}: InstallFormProps) {
  return (
    <form
      className="space-y-6"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <div className="space-y-2">
        <Label htmlFor="install-app-name">App name</Label>
        <Input
          id="install-app-name"
          data-testid="install-app-name-input"
          value={name}
          maxLength={MAX_APP_NAME_LENGTH}
          autoFocus
          onChange={(event) => {
            if (event.target.value.length <= MAX_APP_NAME_LENGTH) {
              onNameChange(event.target.value);
            }
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              onSubmit();
            }
          }}
        />
        {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
      </div>

      {showOrganizationPicker ? (
        <div className="space-y-2">
          <Label htmlFor="install-app-organization">Organization</Label>
          <Select value={organizationId} onValueChange={onOrganizationChange}>
            <SelectTrigger id="install-app-organization" data-testid="install-app-organization-select">
              <SelectValue placeholder="Select an organization" />
            </SelectTrigger>
            <SelectContent>
              {organizations.map((organization) => (
                <SelectItem key={organization.id} value={organization.id}>
                  {organization.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      {installParams && installParams.length > 0 && (
        <InstallParamsForm params={installParams} values={installParamValues} onChange={onInstallParamChange} />
      )}

      <div className="flex flex-row justify-start gap-3 pt-2">
        <LoadingButton
          type="submit"
          data-testid="install-app-submit"
          loading={isInstalling}
          loadingText="Installing..."
          disabled={!name.trim() || !organizationId}
        >
          Install
        </LoadingButton>
      </div>
    </form>
  );
}
