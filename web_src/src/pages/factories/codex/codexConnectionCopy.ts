export const CODEX_CONNECTION_COPY = {
  subscription: "Use your ChatGPT subscription",
  subscriptionDevice: "Open this link and enter the code.",
  subscriptionCopyCode: "Copy sign-in code",
  subscriptionCodeCopied: "Sign-in code copied",
  subscriptionCopyFailed: "SuperPlane could not copy the code. Select the code and copy it.",
  subscriptionDeviceSettings: "In ChatGPT, allow device sign-in in Security settings.",
  subscriptionPending: "Wait for the ChatGPT sign-in code.",
  subscriptionFinishing: "ChatGPT sign-in saved. Finishing connection setup…",
  subscriptionCanceling: "Canceling ChatGPT sign-in. Wait for cleanup to finish.",
  subscriptionStopping: "Wait for the previous connection attempt to stop.",
  subscriptionDelayed:
    "Connection setup is delayed. SuperPlane will keep checking. You can choose another model source.",
  subscriptionReady: "ChatGPT is connected.",
  subscriptionReadyAs: (label: string) => `ChatGPT is connected as ${label}.`,
  subscriptionReconnect: "Connect ChatGPT again.",
  subscriptionCancel: "Cancel sign-in",
  subscriptionBusy: "ChatGPT sign-in is already in progress.",
} as const;
