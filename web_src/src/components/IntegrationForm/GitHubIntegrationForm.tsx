import { useState } from "react";
import { Field, Label, ErrorMessage } from "../Fieldset/fieldset";
import { Input } from "../Input/input";
import { githubConfig } from "./integrationConfigs";
import type { BaseIntegrationFormProps } from "./types";
import githubIcon from "@/assets/icons/integrations/github.svg";

export function GitHubIntegrationForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  orgUrlRef,
}: BaseIntegrationFormProps) {
  const [showGitHubPatInfo, setShowGitHubPatInfo] = useState(false);
  const [dirtyByUser] = useState(false);
  const [displayName, setDisplayName] = useState(() => {
    return integrationData.orgUrl ? githubConfig.extractOrgName(integrationData.orgUrl) : "";
  });

  const handleOrgNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value;
    const sanitizedOrgName = input.replace(/[^a-zA-Z0-9-]/g, "");

    setDisplayName(sanitizedOrgName);

    const url = sanitizedOrgName ? `https://github.com/${sanitizedOrgName}` : "";
    setIntegrationData((prev) => ({ ...prev, orgUrl: url }));

    if (sanitizedOrgName && !dirtyByUser) {
      setIntegrationData((prev) => ({ ...prev, name: `${sanitizedOrgName}-account` }));
    }

    if (errors.orgUrl) {
      setErrors((prev) => ({ ...prev, orgUrl: undefined }));
    }
  };

  return (
    <>
      <Field>
        <Label className="text-sm font-medium text-gray-800 dark:text-white">{githubConfig.orgUrlLabel}</Label>
        <Input
          ref={orgUrlRef}
          type="text"
          data-testid="github-owner-input"
          value={displayName}
          onChange={handleOrgNameChange}
          placeholder={githubConfig.urlPlaceholder}
          className="w-full"
        />
        {errors.orgUrl && <ErrorMessage>{errors.orgUrl}</ErrorMessage>}
        <div className="text-xs text-gray-500 dark:text-gray-400 mt-2">
          <span className="font-semibold">Note:</span> Enter your GitHub organization or username (e.g.,{" "}
          <span className="font-mono">superplanehq</span> from{" "}
          <span className="font-mono">github.com/superplanehq/my_repo</span>).
        </div>
      </Field>

      <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-4">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex-shrink-0">
            <img src={githubIcon} alt="GitHub" className="w-6 h-6" />
          </div>
          <div className="flex-1">
            <div className="text-sm font-medium text-gray-800 dark:text-white">
              Connect GitHub with a Personal Access Token (PAT)
            </div>
            <p className="text-sm text-gray-700 dark:text-gray-300 mt-1">
              To connect GitHub, create a fine-grained Personal Access Token (PAT) and paste it below. You remain in
              full control - you can limit access and revoke it anytime.
            </p>
            <button
              type="button"
              className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
              aria-expanded={showGitHubPatInfo}
              onClick={() => setShowGitHubPatInfo((v) => !v)}
            >
              {showGitHubPatInfo ? "Hide steps" : "Show steps to create your token"}
            </button>
            {showGitHubPatInfo && (
              <div className="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
                <ol className="list-decimal ml-5 mt-1 space-y-1">
                  <li>
                    <a
                      className="text-blue-600 dark:text-blue-400 underline!"
                      href="https://github.com/settings/personal-access-tokens/new"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      Open GitHub to create a new PAT
                    </a>
                  </li>
                  <li>
                    Select the <strong>Resource owner</strong> (has to match the organization/user you entered above)
                  </li>
                  <li>
                    Choose <strong>All repositories</strong> or pick specific repositories
                  </li>
                  <li>
                    Under <strong>Permissions</strong>, set:
                  </li>
                  <ul className="list-disc ml-5 mt-1 space-y-1">
                    <li>
                      <strong>Actions</strong> → Read & Write
                    </li>
                    <li>
                      <strong>Webhooks</strong> → Read & Write
                    </li>
                  </ul>
                  <li>
                    Click <strong>Generate token</strong>, then copy and paste it here
                  </li>
                </ol>
                <p className="text-xs text-gray-600 dark:text-gray-400 mt-2">
                  Tip: You can manage, rotate, or revoke tokens anytime in your{" "}
                  <a
                    className="text-blue-600 dark:text-blue-400 underline!"
                    href="https://github.com/settings/tokens"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    GitHub settings
                  </a>
                  .
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
