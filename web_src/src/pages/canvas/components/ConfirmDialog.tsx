import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: 'danger' | 'primary';
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  isOpen,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  confirmVariant = 'primary',
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-50"
        onClick={onCancel}
      />
      
      {/* Dialog */}
      <div className="relative bg-white dark:bg-zinc-800 rounded-lg shadow-lg max-w-md w-full mx-4 p-6">
        <div className="flex items-start gap-4">
          <div className={`flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center ${
            confirmVariant === 'danger' ? 'text-red-600 bg-red-100' : 'text-blue-600 bg-blue-100'
          }`}>
            <MaterialSymbol 
              name={confirmVariant === 'danger' ? 'warning' : 'help'} 
              size="sm" 
            />
          </div>
          
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-2">
              {title}
            </h3>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-6">
              {message}
            </p>
            
            <div className="flex justify-end gap-3">
              <Button
                outline
                onClick={onCancel}
              >
                {cancelText}
              </Button>
              <Button
                color={confirmVariant === 'danger' ? 'red' : 'blue'}
                onClick={onConfirm}
              >
                {confirmText}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}