interface SidebarHeaderProps {
  stageName: string;
  onClose: () => void;
}

export const SidebarHeader = ({ stageName, onClose }: SidebarHeaderProps) => {
  return (
    <div className="flex items-center justify-between p-6 border-b border-gray-200 bg-gray-50">
      <div className="flex items-center">
        <span className="text-black font-bold mr-2 text-xl">📋</span>
        <span className="text-lg font-bold text-gray-900">{stageName}</span>
      </div>
      <button
        className="text-gray-500 hover:text-gray-700 text-2xl font-bold w-8 h-8 flex items-center justify-center rounded hover:bg-gray-200 transition-colors"
        onClick={onClose}
        title="Close sidebar"
      >
        ×
      </button>
    </div>
  );
};