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
      <div
        onClick={(e) => {
          e.stopPropagation();
          onRevert(sectionId);
        }}
        className="w-5 h-5 rounded-full flex items-center justify-center transition-colors cursor-pointer hover:bg-gray-100"
        title="Revert changes"
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            e.stopPropagation();
            onRevert(sectionId);
          }
        }}
      >
        <MaterialSymbol name="undo" size="sm" className="text-gray-600" />
      </div>
    </div>
  );
}