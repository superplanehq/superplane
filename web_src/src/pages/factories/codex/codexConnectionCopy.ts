export const CODEX_CONNECTION_COPY = {
  subscriptionDevice: "Open ChatGPT and enter this code.",
  subscriptionCopyCode: "Copy sign-in code",
  subscriptionCodeCopied: "Sign-in code copied",
  subscriptionCopyFailed: "SuperPlane could not copy the code. Select the code and copy it.",
  subscriptionDeviceSettings: "In ChatGPT, allow device sign-in in Security settings.",
  subscriptionPending: "Wait for the ChatGPT sign-in code.",
  subscriptionWaiting: "Waiting for ChatGPT sign-in.",
  subscriptionFinishing: "ChatGPT sign-in is saved. Wait for connection setup to finish.",
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
