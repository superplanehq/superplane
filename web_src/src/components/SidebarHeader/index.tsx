interface SidebarHeaderProps {
  stageName: string;
  image: React.ReactNode;
  onClose: () => void;
}

export const SidebarHeader = ({ stageName, image, onClose }: SidebarHeaderProps) => {
  return (
    <div className="flex items-center justify-between px-4 py-3 bg-white dark:bg-zinc-900">
      <div className="flex items-center gap-2">
        {image}
        <span className="text-lg font-semibold text-gray-900 dark:text-zinc-100">{stageName}</span>
      </div>
      <button
        className="p-1 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-200"
        onClick={onClose}
        title="Close sidebar"
      >
        <span className="material-symbols-outlined">close</span>
      </button>
    </div>
  );
};