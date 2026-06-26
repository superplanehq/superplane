import { toast } from "sonner";

export type ToastOptions = {
  id?: string | number;
};

/**
 * Show a standardized error toast notification
 * @param message - The error message to display
 */
export const showErrorToast = (message: string, options?: ToastOptions): void => {
  toast.error(message, options);
};

/**
 * Show a standardized success toast notification
 * @param message - The success message to display
 */
export const showSuccessToast = (message: string, options?: ToastOptions): void => {
  toast.success(message, options);
};

/**
 * Show a standardized informational toast notification (non-error, non-success)
 * @param message - The message to display
 */
export const showInfoToast = (message: string, options?: ToastOptions): void => {
  toast.info(message, options);
};
