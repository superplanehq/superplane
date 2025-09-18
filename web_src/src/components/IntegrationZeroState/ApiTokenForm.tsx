import { Field, Label, ErrorMessage } from '../Fieldset/fieldset';
import { Input } from '../Input/input';
import { ControlledTabs } from '../Tabs/tabs';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel } from '../Dropdown/dropdown';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { Link } from '../Link/link';
import { Text } from '../Text/text';
import type { Tab } from '../Tabs/tabs';
import type { BaseIntegrationFormProps } from './types';

interface ApiTokenFormProps extends BaseIntegrationFormProps {
  organizationId: string;
  canvasId: string;
}

export function ApiTokenForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  apiTokenTab,
  setApiTokenTab,
  newSecretToken,
  setNewSecretToken,
  secrets,
  organizationId,
  canvasId,
}: ApiTokenFormProps) {

  const apiTokenTabs: Tab[] = [
    {
      id: 'new',
      label: 'Create new secret'
    },
    {
      id: 'existing',
      label: 'Import from existing secret',
      disabled: secrets.length === 0,
      disabledTooltip: secrets.length === 0 ? 'No existing secrets found. Please create new one.' : undefined
    }
  ];

  const selectedExistingSecret = secrets.find(secret =>
    secret.metadata?.name === integrationData.apiToken.secretName
  );

  return (
    <div className="space-y-4">
      <div className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-between">
        API Token
      </div>

      <div>
        <ControlledTabs
          tabs={apiTokenTabs}
          activeTab={apiTokenTab}
          variant='pills'
          className='w-full'
          buttonClasses='w-full'
          onTabChange={(tabId) => setApiTokenTab(tabId as 'existing' | 'new')}
        />

        <div className="pt-4">
          {apiTokenTab === 'existing' ? (
            <div className="space-y-4">
              {secrets.length === 0 ? (
                <div className="text-sm text-gray-500 dark:text-zinc-400">
                  No existing secrets found. Please create new one.
                </div>
              ) : (
                <>
                  <Field>
                    <Dropdown>
                      <DropdownButton outline className='flex items-center w-full !justify-between'>
                        {integrationData.apiToken.secretName || 'Select secret'}
                        <MaterialSymbol name="keyboard_arrow_down" />
                      </DropdownButton>
                      <DropdownMenu anchor="bottom start">
                        {secrets.map((secret) => (
                          <DropdownItem
                            key={secret.metadata?.id}
                            onClick={() => {
                              const firstKey = Object.keys(secret.spec?.local?.data || {})[0] || '';
                              setIntegrationData(prev => ({
                                ...prev,
                                apiToken: {
                                  secretName: secret.metadata?.name || '',
                                  secretKey: firstKey
                                }
                              }));
                              if (errors.apiToken) {
                                setErrors(prev => ({ ...prev, apiToken: undefined }));
                              }
                            }}
                          >
                            <DropdownLabel>{secret.metadata?.name}</DropdownLabel>
                          </DropdownItem>
                        ))}
                      </DropdownMenu>
                    </Dropdown>
                  </Field>
                  {selectedExistingSecret && (
                    <Field className='flex items-start gap-3 w-full'>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Key name
                        </Label>
                        <Dropdown>
                          <DropdownButton outline className='flex items-center w-full !justify-between'>
                            {integrationData.apiToken.secretKey || 'Select key'}
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu anchor="bottom start">
                            {Object.keys(selectedExistingSecret.spec?.local?.data || {}).map((key) => (
                              <DropdownItem
                                key={key}
                                onClick={() => setIntegrationData(prev => ({
                                  ...prev,
                                  apiToken: { ...prev.apiToken, secretKey: key }
                                }))}
                              >
                                <DropdownLabel>{key}</DropdownLabel>
                              </DropdownItem>
                            ))}
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Value
                        </Label>
                        <Input
                          type="password"
                          value="••••••••••••••••"
                          readOnly
                          disabled
                          className="w-full bg-gray-50 dark:bg-zinc-800 cursor-not-allowed opacity-75"
                        />
                      </div>
                    </Field>
                  )}
                </>
              )}
              {errors.apiToken && <ErrorMessage>{errors.apiToken}</ErrorMessage>}
            </div>
          ) : (
            <div className="space-y-4 w-full">
              <Text className='text-xs text-gray-500 dark:text-zinc-400'>
                New secret will be created in your canvas secrets.
                You can review and manage your secrets in the secrets tab <Link href={`/${organizationId}/canvas/${canvasId}#secrets`} className='text-blue-600 dark:text-blue-200'>here</Link>
              </Text>

              <Field className='flex items-start gap-3 w-full'>
                <div className='w-full'>
                  <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                    Secret Value
                  </Label>
                  <Input
                    type="password"
                    value={newSecretToken}
                    onChange={(e) => {
                      setNewSecretToken(e.target.value);
                      if (errors.secretValue) {
                        setErrors(prev => ({ ...prev, secretValue: undefined }));
                      }
                    }}
                    placeholder="Enter your API token"
                    className="w-full"
                  />
                  {errors.secretValue && <ErrorMessage>{errors.secretValue}</ErrorMessage>}
                </div>
              </Field>
              {errors.apiToken && <ErrorMessage>{errors.apiToken}</ErrorMessage>}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}