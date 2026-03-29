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
 * Show a standardized info toast notification
 * @param message - The info message to display
 */
export const showInfoToast = (message: string): void => {
  toast.info(message);
};

/**
 * Show a standardized warning toast notification
 * @param message - The warning message to display
 */
export const showWarningToast = (message: string): void => {
  toast.warning(message);
};
