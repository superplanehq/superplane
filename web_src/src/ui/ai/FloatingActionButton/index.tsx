import { BotMessageSquare } from "lucide-react";
import { cn } from "../../../lib/utils";

export namespace FloatingActionButton {
  export interface Props {
    onClick: () => void;
  }
}

export function FloatingActionButton({ onClick }: FloatingActionButton.Props) {
  const buttonClasses = cn(
    "fixed flex items-center justify-center transition-all duration-200 ease-in-out",
    "focus:outline-none",
    "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-md",
    "rounded-lg p-2 px-4",
    "gap-2 hover:gap-3",
    "bg-stone-900",
    "bottom-6 right-6"
  );

  return (
    <button onClick={onClick} className={buttonClasses}>
      <BotMessageSquare className="text-white" size={16} />

      <span className="font-medium whitespace-nowrap text-white text-sm">
        AI Assistant
      </span>
    </button>
  );
}
