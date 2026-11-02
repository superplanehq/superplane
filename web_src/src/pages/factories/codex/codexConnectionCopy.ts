import { getApiErrorMessage } from "@/lib/errors";

export const CODEX_RECONNECT_REQUIRED_MESSAGE =
  "SuperPlane could not refresh your ChatGPT connection. Reconnect ChatGPT to continue.";

const HOSTED_CREDIT_REQUIRED = "This organization has no hosted credit.";

/** Another attempt cannot succeed until an organization admin adds hosted credit. */
export function codexNeedsHostedCredit(message: string): boolean {
  return message.startsWith(HOSTED_CREDIT_REQUIRED);
}

export function codexModelLoadError(error: unknown, connectionStatus?: string): string {
  const detail = getApiErrorMessage(error, "");
  if (codexNeedsHostedCredit(detail)) {
    return detail;
  }
  if (connectionStatus === "reconnect_required" || /connect chatgpt before|reconnect required/i.test(detail)) {
    return CODEX_RECONNECT_REQUIRED_MESSAGE;
  }
  if (/subscription is in use|connection is busy/i.test(detail)) {
    return "Your ChatGPT connection is busy. Try again.";
  }
  return "SuperPlane could not load your ChatGPT models. Try again.";
}

/** A new catalog request can help only while the saved connection stays usable. */
export function codexModelLoadRetryable(message: string, connectionStatus?: string): boolean {
  return (
    connectionStatus === "ready" && !codexNeedsHostedCredit(message) && message !== CODEX_RECONNECT_REQUIRED_MESSAGE
  );
}

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
