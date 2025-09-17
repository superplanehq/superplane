interface ProTipProps {
  show: boolean;
  message?: string;
}

export function ProTip({ show, message = "Pro tip: Type $ to set value from inputs" }: ProTipProps) {
  if (!show) return null;

  return (
    <p className="text-zinc-500 text-xs mb-2 dark:text-zinc-400">
      {message.split('$').map((part, index, array) => (
        <span key={index}>
          {part}
          {index < array.length - 1 && (
            <span className="font-mono bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-zinc-800 dark:text-zinc-200 px-1 py-0.5 rounded">
              $
            </span>
          )}
        </span>
      ))}
    </p>
  );
}