import { toast } from "sonner";

/**
 * Show a standardized error toast notification
 * @param message - The error message to display
 */
export const showErrorToast = (message: string): void => {
  toast.error(message);
};

/**
 * Show a standardized success toast notification
 * @param message - The success message to display
 */
export const showSuccessToast = (message: string): void => {
  toast.success(message);
};

/**
 * Show a standardized informational toast notification (non-error, non-success)
 * @param message - The message to display
 */
export const showInfoToast = (message: string): void => {
  toast.info(message);
};
