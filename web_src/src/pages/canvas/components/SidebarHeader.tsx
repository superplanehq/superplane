interface SidebarHeaderProps {
  stageName: string;
  onClose: () => void;
}

export const SidebarHeader = ({ stageName, onClose }: SidebarHeaderProps) => {
  return (
    <div className="flex items-center justify-between px-4 py-2 bg-white">
      <div className="flex items-center">
        <span className="material-symbols-outlined text-black font-bold mr-1 text-xl">rocket_launch</span>
        <span className="text-normal font-bold text-gray-900">{stageName}</span>
      </div>
      <button
        className="text-black text-[30px] flex items-center justify-center rounded "
        onClick={onClose}
        title="Close sidebar"
      >
        ×
      </button>
    </div>
  );
};