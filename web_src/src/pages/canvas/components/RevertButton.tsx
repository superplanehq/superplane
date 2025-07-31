import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface RevertButtonProps {
  sectionId: string;
  isModified: boolean;
  onRevert: (sectionId: string) => void;
}

export function RevertButton({ sectionId, isModified, onRevert }: RevertButtonProps) {
  if (!isModified) {
    return null;
  }

  return (
    <div className="flex items-center gap-1">
      <div className="w-1.5 h-1.5 bg-orange-500 rounded-full"></div>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onRevert(sectionId);
        }}
        className="w-5 h-5 rounded-full flex items-center justify-center transition-colors"
        title="Revert changes"
      >
        <MaterialSymbol name="undo" size="sm" className="text-gray-600" />
      </button>
    </div>
  );
}