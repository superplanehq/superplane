import { useState } from 'react';
import { Field, Label, ErrorMessage } from '../Fieldset/fieldset';
import { Input } from '../Input/input';
import { semaphoreConfig } from './integrationConfigs';
import type { BaseIntegrationFormProps } from './types';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';

export function SemaphoreIntegrationForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  orgUrlRef
}: BaseIntegrationFormProps) {
  const [showSemaphoreTokenInfo, setShowSemaphoreTokenInfo] = useState(false);
  const [dirtyByUser, setDirtyByUser] = useState(false);

  const handleOrgUrlChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const url = e.target.value;
    setIntegrationData(prev => ({ ...prev, orgUrl: url }));

    if (url && !dirtyByUser) {
      const orgName = semaphoreConfig.extractOrgName(url);
      if (orgName) {
        setIntegrationData(prev => ({ ...prev, name: `${orgName}-organization` }));
      }
    }

    if (errors.orgUrl) {
      setErrors(prev => ({ ...prev, orgUrl: undefined }));
    }
  };

  return (
    <>
      <Field>
        <Label className="text-sm font-medium text-gray-900 dark:text-white">
          {semaphoreConfig.orgUrlLabel}
        </Label>
        <Input
          ref={orgUrlRef}
          type="url"
          value={integrationData.orgUrl}
          onChange={handleOrgUrlChange}
          placeholder={semaphoreConfig.urlPlaceholder}
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
            <div className="text-sm font-medium text-gray-900 dark:text-white">Connect Semaphore with your API Token</div>
            <p className="text-sm text-zinc-700 dark:text-zinc-300 mt-1">
              To connect Semaphore, create an API Token in your account settings and paste it below.
            </p>
            <button
              type="button"
              className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
              aria-expanded={showSemaphoreTokenInfo}
              onClick={() => setShowSemaphoreTokenInfo(v => !v)}
            >
              {showSemaphoreTokenInfo ? 'Hide steps' : 'Show steps to create your token'}
            </button>
            {showSemaphoreTokenInfo && (
              <div className="mt-3 space-y-2 text-sm text-zinc-700 dark:text-zinc-300">
                <ol className="list-decimal ml-5 mt-1 space-y-1">
                  <li>
                    You can find your token by visiting your <a className="text-blue-600 dark:text-blue-400 underline!" href="https://me.semaphoreci.com/account" target="_blank" rel="noopener noreferrer">account settings</a>.
                  </li>
                  <li>Click on <strong>Regenerate API Token</strong></li>
                  <li>Check if your Role has the <a className="text-blue-600 dark:text-blue-400 underline!" href="https://docs.semaphore.io/using-semaphore/rbac#org-member" target="_blank" rel="noopener noreferrer">required permissions</a> in the organization:</li>
                  <ul className="list-disc ml-5 mt-1 space-y-1">
                    <li><strong>Secrets</strong> → View and Manage</li>
                    <li><strong>Notifications</strong> → View and Manage</li>
                  </ul>
                  <li>Copy and paste the token here</li>
                </ol>
                <p className="text-xs text-zinc-600 dark:text-zinc-400 mt-2">
                  Tip: You can check your current organization role at the People page of your organization. Check the <a className="text-blue-600 dark:text-blue-400 underline!" href="https://docs.semaphore.io/using-semaphore/rbac" target="_blank" rel="noopener noreferrer">Role Based Access Control (RBAC)</a> documentation for more information.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}