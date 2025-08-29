import { useState } from 'react';
import { Field, Label, ErrorMessage } from '../Fieldset/fieldset';
import { Input } from '../Input/input';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { githubConfig } from './integrationConfigs';
import type { BaseIntegrationFormProps } from './types';


export function GitHubIntegrationForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  orgUrlRef
}: BaseIntegrationFormProps) {
  const [showGitHubPatInfo, setShowGitHubPatInfo] = useState(false);
  const [dirtyByUser, setDirtyByUser] = useState(false);
  const [displayName, setDisplayName] = useState(() => {
    return integrationData.orgUrl ? githubConfig.extractOrgName(integrationData.orgUrl) : '';
  });

  const handleOrgNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value;
    const sanitizedOrgName = input.replace(/[^a-zA-Z0-9-]/g, '');

    setDisplayName(sanitizedOrgName);

    const url = sanitizedOrgName ? `https://github.com/${sanitizedOrgName}` : '';
    setIntegrationData(prev => ({ ...prev, orgUrl: url }));

    if (sanitizedOrgName && !dirtyByUser) {
      setIntegrationData(prev => ({ ...prev, name: `${sanitizedOrgName}-account` }));
    }

    if (errors.orgUrl) {
      setErrors(prev => ({ ...prev, orgUrl: undefined }));
    }
  };

  return (
    <>
      <Field>
        <Label className="text-sm font-medium text-gray-900 dark:text-white">
          {githubConfig.orgUrlLabel}
        </Label>
        <Input
          ref={orgUrlRef}
          type="text"
          value={displayName}
          onChange={handleOrgNameChange}
          placeholder={githubConfig.urlPlaceholder}
          className="w-full"
        />
        {errors.orgUrl && <ErrorMessage>{errors.orgUrl}</ErrorMessage>}
      </Field>

      <Field>
        <Label className="text-sm font-medium text-gray-900 dark:text-white">
          Integration Name
        </Label>
        <Input
          type="text"
          value={integrationData.name}
          onChange={(e) => {
            setIntegrationData(prev => ({ ...prev, name: e.target.value }));
            if (errors.name) {
              setErrors(prev => ({ ...prev, name: undefined }));
            }

            if (e.target.value === '') {
              setDirtyByUser(false);
            }
          }}
          onKeyDown={() => setDirtyByUser(true)}
          placeholder="Enter integration name"
          className="w-full"
        />
        {errors.name && <ErrorMessage>{errors.name}</ErrorMessage>}
      </Field>

      <div className="rounded-md border border-gray-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 p-4">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 text-zinc-600 dark:text-zinc-300">
            <MaterialSymbol name="info" size="md" />
          </div>
          <div className="flex-1">
            <div className="text-sm font-medium text-gray-900 dark:text-white">GitHub Personal Access Token (PAT) required</div>
            <p className="text-sm text-zinc-700 dark:text-zinc-300 mt-1">
              To connect GitHub, create a fine‑grained Personal Access Token and provide it as the API token.
            </p>
            <button
              type="button"
              className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
              aria-expanded={showGitHubPatInfo}
              onClick={() => setShowGitHubPatInfo(v => !v)}
            >
              {showGitHubPatInfo ? 'Hide details' : 'Show how to configure PAT'}
            </button>
            {showGitHubPatInfo && (
              <div className="mt-3 space-y-2 text-sm text-zinc-700 dark:text-zinc-300">
                <p>When creating a fine‑grained PAT</p>
                <div><strong>Chose the access scope:</strong></div>
                <ul className="list-disc ml-5 mt-1 space-y-1">
                  <li>All repositories</li>
                  <li>Or select specific repositories</li>
                </ul>
                <div className="mt-2"><strong>Set required permissions:</strong></div>
                <ul className="list-disc ml-5 mt-1 space-y-1">
                  <li>Actions - Read AND Write</li>
                  <li>Webhooks - Read AND Write</li>
                </ul>
                <p className="text-xs text-zinc-600 dark:text-zinc-400">
                  Tip: You can manage or rotate the PAT anytime in your GitHub developer settings.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}