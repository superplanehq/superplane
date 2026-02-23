export interface AmiStateChangeDetail {
  ImageId?: string;
  State?: string;
  ErrorMessage?: string;
}

export interface AmiStateChangeEvent {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: AmiStateChangeDetail;
}

export interface Ec2Image {
  imageId?: string;
  name?: string;
  description?: string;
  state?: string;
  creationDate?: string;
  ownerId?: string;
  architecture?: string;
  imageType?: string;
  rootDeviceType?: string;
  rootDeviceName?: string;
  virtualizationType?: string;
  hypervisor?: string;
}
