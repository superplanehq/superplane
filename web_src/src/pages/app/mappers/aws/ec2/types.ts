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

export interface Ec2Instance {
  instanceId?: string;
  instanceType?: string;
  imageId?: string;
  state?: string;
  name?: string;
  keyName?: string;
  launchTime?: string;
  privateIpAddress?: string;
  publicIpAddress?: string;
  privateDnsName?: string;
  publicDnsName?: string;
  subnetId?: string;
  vpcId?: string;
  region?: string;
}

export interface ElbLoadBalancer {
  loadBalancerArn?: string;
  name?: string;
  dnsName?: string;
  scheme?: string;
  type?: string;
  state?: string;
  vpcId?: string;
  region?: string;
}
