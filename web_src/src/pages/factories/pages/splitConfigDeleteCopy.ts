import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";

export interface SplitConfigDeleteCopy {
  menuLabel: string;
  title: string;
  description: string;
  confirmLabel: string;
  pendingLabel: string;
  keepLabel: string;
}

export const DEFAULT_SPLIT_CONFIG_DELETE_COPY: SplitConfigDeleteCopy = {
  menuLabel: COLUMN_AUTOMATIONS_COPY.deleteLabel,
  title: COLUMN_AUTOMATIONS_COPY.deleteLabel,
  description: COLUMN_AUTOMATIONS_COPY.confirmDelete,
  confirmLabel: COLUMN_AUTOMATIONS_COPY.deleteLabel,
  pendingLabel: COLUMN_AUTOMATIONS_COPY.deletingLabel,
  keepLabel: COLUMN_AUTOMATIONS_COPY.keepLabel,
};
