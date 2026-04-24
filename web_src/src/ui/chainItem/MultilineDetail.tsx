type MultilineDetailProps = {
  label: string;
  value: string;
};

export function MultilineDetail({ label, value }: MultilineDetailProps) {
  return (
    <div className="flex flex-col gap-1 px-2 rounded-md w-full min-w-0 font-medium">
      <span className="text-[13px] text-left text-gray-600" title={label}>
        {label}:
      </span>
      <pre className="text-[12px] font-mono bg-gray-50 border border-gray-200 rounded p-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-gray-800 w-full">
        {value}
      </pre>
    </div>
  );
}
