interface SidebarHeaderProps {
  stageName: string;
  image: React.ReactNode;
  onClose: () => void;
}

export const SidebarHeader = ({ stageName, image, onClose }: SidebarHeaderProps) => {
  return (
    <div className="flex items-center justify-between px-4 py-3 bg-white">
      <div className="flex items-center gap-2">
        {image}
        <span className="text-lg font-semibold text-gray-900">{stageName}</span>
      </div>
      <button
        className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
        onClick={onClose}
        title="Close sidebar"
      >
        <span className="material-symbols-outlined">close</span>
      </button>
    </div>
  );
};