import type { FactoriesWorkOrder } from "@/api-client";

export function formatWorkOrderState(state?: FactoriesWorkOrder["state"]) {
  switch (state) {
    case "STATE_DRAFT":
      return "Draft";
    case "STATE_OPEN":
      return "Open";
    case "STATE_CLOSED":
      return "Closed";
    default:
      return "Unknown";
  }
}

export function formatWorkOrderResult(result?: FactoriesWorkOrder["result"]) {
  switch (result) {
    case "RESULT_COMPLETED":
      return "Completed";
    case "RESULT_REJECTED":
      return "Rejected";
    case "RESULT_FAILED":
      return "Failed";
    default:
      return "";
  }
}

export function formatTimestamp(value?: string) {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}
