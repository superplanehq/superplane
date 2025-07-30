interface StageButtonProps {
  children: React.ReactNode;
  className?: string;
  color?: "blue" | "zinc";
  plain?: boolean;
  onClick?: () => void;
  type?: "button" | "submit" | "reset";
}

export function StageButton({ 
  children, 
  className = "", 
  color = "zinc", 
  plain = false, 
  onClick,
  type = "button"
}: StageButtonProps) {
  const baseClasses = "px-3 py-2 text-sm font-medium rounded-md transition-colors flex items-center gap-2";
  const colorClasses = {
    blue: "bg-blue-600 hover:bg-blue-700 text-white",
    zinc: "bg-zinc-100 hover:bg-zinc-200 text-zinc-900 dark:bg-zinc-700 dark:hover:bg-zinc-600 dark:text-zinc-100"
  };
  const plainClasses = "text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200 bg-transparent hover:bg-zinc-100 dark:hover:bg-zinc-800";

  return (
    <button
      type={type}
      className={`${baseClasses} ${plain ? plainClasses : colorClasses[color]} ${className}`}
      onClick={onClick}
    >
      {children}
    </button>
  );
}