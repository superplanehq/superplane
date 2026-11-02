import type { FleetDraft, FleetSection } from "@/lib/organizationFleetConfiguration";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";

export function FleetConfigurationFields({
  section,
  draft,
  onChange,
  disabled,
}: {
  section: FleetSection | "identity";
  draft: FleetDraft;
  onChange: (draft: FleetDraft) => void;
  disabled?: boolean;
}) {
  if (section === "platform") return <PlatformFields draft={draft} onChange={onChange} disabled={disabled} />;
  const fields = {
    identity: [{ key: "fleetId", label: "Fleet ID", placeholder: "my-fleet" }],
    version: [{ key: "runnerVersion", label: "Runner version", placeholder: "v1.2.3" }],
    resources: [
      { key: "cpuMillicores", label: "CPU millicores", placeholder: "1000" },
      { key: "memoryMb", label: "Memory MB", placeholder: "1024" },
      { key: "diskGb", label: "Disk GB", placeholder: "20" },
      { key: "capabilities", label: "Capabilities (comma-separated)", placeholder: "docker, gpu" },
    ],
  } as const;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {fields[section].map(({ key, label, placeholder }) => {
        const numeric = key === "cpuMillicores" || key === "memoryMb" || key === "diskGb";
        return (
          <div key={key} className={section === "identity" || section === "version" ? "sm:col-span-2" : ""}>
            <Label htmlFor={`fleet-${key}`} className="mb-1.5 block text-[12px]">
              {label}
            </Label>
            <Input
              id={`fleet-${key}`}
              value={draft[key]}
              disabled={disabled}
              required={key !== "capabilities"}
              type={numeric ? "number" : "text"}
              min={numeric ? 1 : undefined}
              step={numeric ? 1 : undefined}
              placeholder={placeholder}
              className="text-[13px]"
              onChange={(event) => onChange({ ...draft, [key]: event.target.value })}
            />
          </div>
        );
      })}
    </div>
  );
}

function PlatformFields({
  draft,
  onChange,
  disabled,
}: {
  draft: FleetDraft;
  onChange: (draft: FleetDraft) => void;
  disabled?: boolean;
}) {
  const fields = [
    { key: "operatingSystem", label: "Operating system", options: [{ value: "linux", label: "Linux" }] },
    {
      key: "architecture",
      label: "Architecture",
      options: [
        { value: "amd64", label: "x86-64 (amd64)" },
        { value: "arm64", label: "ARM64 (arm64)" },
      ],
    },
  ] as const;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {fields.map(({ key, label, options }) => (
        <div key={key}>
          <Label htmlFor={`fleet-${key}`} className="mb-1.5 block text-[12px]">
            {label}
          </Label>
          <Select
            value={draft[key]}
            disabled={disabled}
            required
            onValueChange={(value) => onChange({ ...draft, [key]: value })}
          >
            <SelectTrigger id={`fleet-${key}`} className="w-full text-[13px]">
              <SelectValue placeholder={`Select ${label.toLowerCase()}`} />
            </SelectTrigger>
            <SelectContent>
              {draft[key] && !options.some((option) => option.value === draft[key]) && (
                <SelectItem value={draft[key]}>{draft[key]}</SelectItem>
              )}
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ))}
    </div>
  );
}
