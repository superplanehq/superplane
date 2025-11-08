import { toast } from "react-toastify";

/**
 * Show a standardized error toast notification
 * @param message - The error message to display
 */
export const showErrorToast = (message: string): void => {
  toast.error(message, {
    position: "bottom-center",
    autoClose: 5000,
    hideProgressBar: false,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    closeButton: false,
    className: "custom-toast",
  });
};

/**
 * Show a standardized success toast notification
 * @param message - The success message to display
 */
export const showSuccessToast = (message: string): void => {
  toast.success(message, {
    position: "bottom-center",
    autoClose: 5000,
    hideProgressBar: false,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    closeButton: false,
    className: "custom-toast",
  });
};

/**
 * Show a standardized info toast notification
 * @param message - The info message to display
 */
export const showInfoToast = (message: string): void => {
  toast.info(message, {
    position: "bottom-center",
    autoClose: 5000,
    hideProgressBar: false,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    closeButton: false,
    className: "custom-toast",
  });
};

/**
 * Show a standardized warning toast notification
 * @param message - The warning message to display
 */
export const showWarningToast = (message: string): void => {
  toast.warning(message, {
    position: "bottom-center",
    autoClose: 5000,
    hideProgressBar: false,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    closeButton: false,
    className: "custom-toast",
  });
};
