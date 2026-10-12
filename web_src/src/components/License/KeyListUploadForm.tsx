import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { uploadLicenseKeyList, type InstallationLicense } from "@/lib/license";
import React, { useState } from "react";

type KeyListUploadFormProps = {
  onUploaded: (license: InstallationLicense) => void;
};

export const KeyListUploadForm: React.FC<KeyListUploadFormProps> = ({ onUploaded }) => {
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setUploading(true);

    try {
      const license = await uploadLicenseKeyList(value);
      setValue("");
      onUploaded(license);
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : "Failed to upload the key list");
    } finally {
      setUploading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <Label htmlFor="license-key-list" className="mb-2 block text-left">
          Key list
        </Label>
        <Textarea
          id="license-key-list"
          data-testid="license-key-list-input"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          placeholder="Paste the key list that you received from SuperPlane"
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

      <Button type="submit" data-testid="license-key-list-upload" disabled={uploading || value.trim() === ""}>
        {uploading ? "Verifying..." : "Upload key list"}
      </Button>
    </form>
  );
};
