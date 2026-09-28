export const DATADOG_INTAKE_SETTINGS_COPY = {
  serviceLabel: "Service",
  intakeSection: "Create task when:",
  triggered: "A new error issue is reported",
  environmentsLabel: "Environments",
  environmentsHelper: "Select none to keep every environment.",
  environmentsLoading: "Loading environments.",
  environmentsError: "The application key cannot read environments. Add apm_read or logs_read_data.",
  environmentsEmpty: "No environments for this service.",
} as const;
