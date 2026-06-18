export type SettingsValues = {
  name: string;
  description: string;
};

export type SettingsSavePayload = {
  name: string;
  description: string;
};

export interface SettingsViewProps {
  initialValues: SettingsValues;
  canUpdateCanvas: boolean;
  isSaving: boolean;
  onSave: (values: SettingsSavePayload) => Promise<void>;
  onBackToCanvas?: () => void;
}
