import { MaterialSymbol } from "../MaterialSymbol/material-symbol";
import React, { useState, useEffect, createContext, useContext } from "react";

interface Toast {
  id: string;
  type: ToastType;
  title: string;
  description: string;
}

interface ToastConfig {
  iconName: string;
  iconColor: string;
  bgColor: string;
}

type ToastType = "error" | "success" | "info";

interface ToastContextType {
  addToast: (type: ToastType, title: string, description: string) => void;
}

const ToastContext = createContext<ToastContextType | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = (type: ToastType, title: string, description: string) => {
    const toast: Toast = {
      id: Math.random().toString(36).substring(7),
      type,
      title,
      description,
    };

    setToasts(prev => [...prev, toast]);

    // Auto remove after 4 seconds
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== toast.id));
    }, 4000);
  };

  const removeToast = (id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  };

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <ToasterBar toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  );
}

function ToasterBar({ toasts, onRemove }: { toasts: Toast[]; onRemove: (id: string) => void }) {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2 pointer-events-none">
      {toasts.map(toast => (
        <ToastComponent key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  );
}

function ToastComponent({ toast, onRemove }: { toast: Toast; onRemove: (id: string) => void }) {
  const config = toastConfigs[toast.type];
  
  return (
    <div 
      className={`${config.bgColor} border border-gray-200 dark:border-zinc-700 p-3 rounded-lg shadow-lg text-xs pointer-events-auto transition-all duration-300 max-w-sm`}
      onClick={() => onRemove(toast.id)}
    >
      <div className="flex gap-2">
        <MaterialSymbol name={config.iconName} size="sm" className={`${config.iconColor} mt-0.5 flex-shrink-0`} />
        <div className="flex-1">
          <div className="font-semibold text-gray-900 dark:text-white">{toast.title}</div>
          <div className="text-gray-600 dark:text-gray-300">{toast.description}</div>
        </div>
      </div>
    </div>
  );
}

const toastConfigs: Record<ToastType, ToastConfig> = {
  error: {
    iconName: "warning",
    iconColor: "text-red-600 dark:text-red-400",
    bgColor: "bg-red-50 dark:bg-red-900/20",
  },
  success: {
    iconName: "check_circle",
    iconColor: "text-green-600 dark:text-green-400", 
    bgColor: "bg-green-50 dark:bg-green-900/20",
  },
  info: {
    iconName: "info",
    iconColor: "text-blue-600 dark:text-blue-400",
    bgColor: "bg-blue-50 dark:bg-blue-900/20",
  },
};

// Hook to use toasts
export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
}

// Global toast functions for backward compatibility
let globalToastFunction: ((type: ToastType, title: string, description: string) => void) | null = null;

export function setGlobalToastFunction(fn: (type: ToastType, title: string, description: string) => void) {
  globalToastFunction = fn;
}

export const showErrorToast = (title: string, description: string) => {
  if (globalToastFunction) {
    globalToastFunction("error", title, description);
  }
};

export const showSuccessToast = (title: string, description: string) => {
  if (globalToastFunction) {
    globalToastFunction("success", title, description);
  }
};

export const showInfoToast = (title: string, description: string) => {
  if (globalToastFunction) {
    globalToastFunction("info", title, description);
  }
};