import { useState } from 'react';
import { Field, Label, ErrorMessage } from '../Fieldset/fieldset';
import { Input } from '../Input/input';
import { semaphoreConfig } from './integrationConfigs';
import type { BaseIntegrationFormProps } from './types';

export function SemaphoreIntegrationForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  orgUrlRef
}: BaseIntegrationFormProps) {
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
    </>
  );
}