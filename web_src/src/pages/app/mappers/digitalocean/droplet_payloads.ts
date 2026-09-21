export type DropletPayload = {
  id?: number | string;
  name?: string;
  status?: string;
  size_slug?: string;
  memory?: number | string;
  vcpus?: number | string;
  disk?: number | string;
  tags?: string[];
  region?: { name?: string; slug?: string };
  image?: { name?: string; slug?: string };
  networks?: { v4?: { type?: string; ip_address?: string }[] };
};

export type SnapshotPayload = {
  id?: number | string;
  name?: string;
  resource_id?: number | string;
  regions?: string[];
  min_disk_size?: number | string;
  size_gigabytes?: number | string;
};

export type DeleteDropletResult = {
  dropletId?: number | string;
};

export type DeleteSnapshotResult = {
  snapshotId?: number | string;
  deleted?: boolean;
};
