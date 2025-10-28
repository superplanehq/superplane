import { BotMessageSquare, SparklesIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "../../../lib/utils";

export namespace FloatingActionButton {
  export interface Props {
    onClick: () => void;
  }
}

export function FloatingActionButton({ onClick }: FloatingActionButton.Props) {
  const buttonClasses = cn(
    "flex items-center justify-center transition-all duration-200 ease-in-out",
    "focus:outline-none",
    "rounded-lg p-2 px-4",
    "gap-2 hover:gap-3",
    "bg-stone-900"
  );

  const [scale, setScale] = useState(0.9);

  useEffect(() => {
    const interval = setInterval(() => {
      setScale((prev) => (prev === 0.9 ? 1.1 : 0.9));
    }, 500);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="fixed bottom-6 right-6 flex items-center gap-2">
      <div className="flex items-center select-none">
        <div
          className={cn(
            "relative overflow-hidden",
            "border-2 border-yellow-300 bg-white text-sm px-3 py-2 rounded-md shadow-md",
            "flex items-center gap-3"
          )}
        >
          <SparklesIcon
            className="text-yellow-400 transition"
            style={{ scale: scale, rotate: `${(scale - 0.9) * 50}deg` }}
            size={16}
          />{" "}
          3 improvements available
        </div>

        <div className="w-0 h-0 border-t-8 border-b-8 border-l-8 border-t-transparent border-b-transparent border-l-yellow-300 drop-shadow" />
      </div>

      <button onClick={onClick} className={buttonClasses}>
        <BotMessageSquare className="text-white" size={16} />
        <span className="text-white text-sm">AI Assistant</span>
      </button>
    </div>
  );
}
