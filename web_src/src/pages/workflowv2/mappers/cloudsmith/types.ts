export interface OnPackageEventMetadata {
  repository?: string;
  webhookSlug?: string;
  webhookUrl?: string;
}

export interface PackageInfo {
  name?: string;
  version?: string;
  format?: string;
  size?: number;
  checksum_sha256?: string;
  cdn_url?: string;
  status_str?: string;
}
