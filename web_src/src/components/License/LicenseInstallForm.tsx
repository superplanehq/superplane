import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { installInstallationLicense, type InstallationLicense } from "@/lib/license";
import { cn } from "@/lib/utils";
import { FileText, Upload, X } from "lucide-react";
import React, { useRef, useState } from "react";

const MAX_LICENSE_FILE_BYTES = 64 * 1024;

type SelectedLicenseFile = {
  name: string;
  size: number;
  text: string;
};

type LicenseInstallFormProps = {
  submitLabel: string;
  onInstalled: (license: InstallationLicense) => void;
  onInstallingChange?: (installing: boolean) => void;
  secondaryAction?: React.ReactNode;
};

const formatFileSize = (bytes: number) => (bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(1)} KB`);

type LicenseFileFieldProps = {
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  selectedFile: SelectedLicenseFile | null;
  dragging: boolean;
  onDragging: (dragging: boolean) => void;
  onFile: (file: File | undefined) => void;
  onClear: () => void;
};

const LicenseFileField: React.FC<LicenseFileFieldProps> = ({
  fileInputRef,
  selectedFile,
  dragging,
  onDragging,
  onFile,
  onClear,
}) => (
  <div>
    <Label htmlFor="license-file" className="mb-2 block text-left">
      License file
    </Label>
    <label
      htmlFor="license-file"
      data-testid="license-file-dropzone"
      onDragOver={(event) => {
        event.preventDefault();
        onDragging(true);
      }}
      onDragLeave={() => onDragging(false)}
      onDrop={(event) => {
        event.preventDefault();
        onDragging(false);
        onFile(event.dataTransfer.files[0]);
      }}
      className={cn(
        "flex cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed px-4 py-8 text-center transition-colors",
        dragging
          ? "border-gray-900 bg-slate-100 dark:border-gray-200 dark:bg-gray-800"
          : "border-slate-300 bg-slate-50 hover:bg-slate-100 dark:border-gray-600 dark:bg-gray-800/50 dark:hover:bg-gray-800",
      )}
    >
      <Upload className="mb-2 h-5 w-5 text-gray-500 dark:text-gray-400" />
      <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
        Drop your license file here, or browse
      </span>
      <span className="mt-1 text-xs text-gray-500 dark:text-gray-400">.license or text file, up to 64 KB</span>
    </label>
    <Input
      ref={fileInputRef}
      id="license-file"
      data-testid="license-file-input"
      type="file"
      accept=".license,.txt,.jwt,text/plain"
      className="sr-only"
      onChange={(event) => {
        onFile(event.target.files?.[0]);
        event.target.value = "";
      }}
    />
    {selectedFile ? (
      <div
        className="mt-3 flex items-center justify-between gap-3 rounded-md border border-slate-200 px-3 py-2 dark:border-gray-700"
        data-testid="license-file-selected"
      >
        <div className="flex min-w-0 items-center gap-2">
          <FileText className="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" />
          <div className="min-w-0">
            <p className="truncate text-sm text-gray-900 dark:text-gray-100">{selectedFile.name}</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">{formatFileSize(selectedFile.size)}</p>
          </div>
        </div>
        <button
          type="button"
          aria-label="Remove license file"
          className="rounded-md p-1 text-gray-500 hover:bg-slate-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
          onClick={onClear}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    ) : null}
  </div>
);

const readLicenseFile = async (file: File): Promise<SelectedLicenseFile> => {
  if (file.size > MAX_LICENSE_FILE_BYTES) {
    throw new Error("Choose a license file smaller than 64 KB.");
  }

  const text = (await file.text()).trim();
  if (text === "") {
    throw new Error("That license file is empty.");
  }

  return { name: file.name, size: file.size, text };
};

export const LicenseInstallForm: React.FC<LicenseInstallFormProps> = ({
  submitLabel,
  onInstalled,
  onInstallingChange,
  secondaryAction,
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const choiceRef = useRef(0);
  const [pasted, setPasted] = useState("");
  const [selectedFile, setSelectedFile] = useState<SelectedLicenseFile | null>(null);
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [installing, setInstalling] = useState(false);
  const licenseText = (selectedFile?.text ?? pasted).trim();

  const replaceChoice = () => {
    choiceRef.current += 1;
  };

  const acceptFile = async (file: File | undefined) => {
    if (!file) {
      return;
    }

    const choice = choiceRef.current + 1;
    choiceRef.current = choice;
    setError(null);
    try {
      const nextFile = await readLicenseFile(file);
      if (choice !== choiceRef.current) return;
      setSelectedFile(nextFile);
      setPasted("");
    } catch (readError) {
      if (choice !== choiceRef.current) return;
      setSelectedFile(null);
      setError(readError instanceof Error ? readError.message : "Could not read the license file.");
    }
  };

  const clearFile = () => {
    setSelectedFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setInstalling(true);
    onInstallingChange?.(true);

    try {
      const license = await installInstallationLicense(licenseText);
      setPasted("");
      clearFile();
      onInstalled(license);
    } catch (installError) {
      setError(installError instanceof Error ? installError.message : "Failed to install the license");
    } finally {
      setInstalling(false);
      onInstallingChange?.(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <LicenseFileField
        fileInputRef={fileInputRef}
        selectedFile={selectedFile}
        dragging={dragging}
        onDragging={setDragging}
        onFile={(file) => {
          void acceptFile(file);
        }}
        onClear={clearFile}
      />

      <div className="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
        <div className="h-px flex-1 bg-gray-300 dark:bg-gray-700" />
        <span>or paste the key</span>
        <div className="h-px flex-1 bg-gray-300 dark:bg-gray-700" />
      </div>

      <div>
        <Label htmlFor="license-key" className="sr-only">
          License key
        </Label>
        <Textarea
          id="license-key"
          data-testid="license-key-input"
          value={pasted}
          onChange={(event) => {
            replaceChoice();
            setPasted(event.target.value);
            clearFile();
          }}
          placeholder="Paste the license key that you received from SuperPlane"
          spellCheck={false}
          autoComplete="off"
          aria-invalid={error ? true : undefined}
          className="min-h-20 font-mono text-xs break-all"
        />
        {error ? (
          <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
            {error}
          </p>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" data-testid="license-install" disabled={installing || licenseText === ""}>
          {installing ? "Verifying..." : submitLabel}
        </Button>
        {secondaryAction}
      </div>
    </form>
  );
};
