import { BotMessageSquare, SparklesIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "../../../lib/utils";

export namespace FloatingActionButton {
  export interface Props {
    onClick: () => void;

    showNotification?: boolean;
    notificationMessage?: string;
  }
}

export function FloatingActionButton(props: FloatingActionButton.Props) {
  const buttonClasses = cn(
    "flex items-center justify-center transition-all duration-200 ease-in-out",
    "focus:outline-none",
    "rounded-lg p-2 px-4",
    "gap-2 hover:gap-3",
    "bg-stone-900"
  );

  return (
    <div className="fixed bottom-6 right-6 flex items-center gap-2">
      <NotificationBubble
        show={props.showNotification}
        message={props.notificationMessage}
      />

      <button onClick={props.onClick} className={buttonClasses}>
        <BotMessageSquare className="text-white" size={16} />
        <span className="text-white text-sm">AI Assistant</span>
      </button>
    </div>
  );
}

function NotificationBubble({
  show,
  message,
}: {
  show?: boolean;
  message?: string;
}) {
  if (!show) {
    return null;
  }

  return (
    <div className="flex items-center select-none">
      <div
        className={cn(
          "relative overflow-hidden",
          "border-2 border-yellow-300 bg-white text-sm px-3 py-1.5 rounded-md shadow-md",
          "flex items-center gap-3"
        )}
      >
        <PulsatingSparklesIcon />
        <span className="font-medium text-stone-900">{message}</span>
      </div>

      <div className="w-0 h-0 border-t-8 border-b-8 border-l-8 border-t-transparent border-b-transparent border-l-yellow-300 drop-shadow" />
    </div>
  );
}

function PulsatingSparklesIcon() {
  const [scale, setScale] = useState(0.9);

  useEffect(() => {
    const interval = setInterval(() => {
      setScale((prev) => (prev === 0.9 ? 1.1 : 0.9));
    }, 500);

    return () => clearInterval(interval);
  }, []);

  return (
    <SparklesIcon
      className="text-yellow-400 transition-all"
      style={{ scale: scale, rotate: `${(scale - 0.9) * 50}deg` }}
      size={16}
    />
  );
}
