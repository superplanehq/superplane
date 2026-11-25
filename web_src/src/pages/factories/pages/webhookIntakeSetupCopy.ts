export const WEBHOOK_INTAKE_SETUP_COPY = {
  pageTitle: "How should webhook requests become tasks?",
  wizardStepWebhook: "Choose a webhook",
  wizardStepWebhookHelper: "Other tools send requests to this webhook. Each request becomes a task in Backlog.",
  wizardStepFields: "Which request fields fill the task?",
  wizardStepFieldsHelper:
    "Enter the JSON body field for each task field. Use dots for nested fields, for example issue.title.",
  modeNew: "Create a webhook",
  modeExisting: "Use an existing webhook",
  nameLabel: "Webhook name",
  defaultName: "Webhook intake",
  existingHelper: "Each request to this webhook becomes a task.",
  newWebhookHelper: "SuperPlane shows the URL and the secret after it creates the intake.",
  wizardContinue: "Continue",
  wizardFinish: "Finish",
  wizardFinishing: "Finishing...",
  webhookCreateError: "SuperPlane could not create the webhook.",
  wizardCreateError: "SuperPlane could not create the webhook intake.",
} as const;

export const WEBHOOK_INTAKE_FIELDS_COPY = {
  titleLabel: "Title field",
  titleHelper: 'A request without this field gets the title "Webhook request".',
  descriptionLabel: "Description field",
  descriptionHelper: "Leave empty to create tasks without a description.",
  linkLabel: "Link field",
  linkHelper:
    "SuperPlane skips a request when a task with the same link exists. Leave empty to create a task for each request.",
  exampleLabel: "Example request body",
  titleRequired: "Enter the title field.",
  pathInvalid: "Use names with letters, digits, - or _, separated by dots.",
} as const;
