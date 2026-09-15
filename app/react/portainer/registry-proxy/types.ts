export type RegistryProxyId = number;

export interface RegistryProxy {
  Id: RegistryProxyId;
  Name: string;
  URL: string;
  TLS: boolean;
  TLSSkipVerify: boolean;
  Authentication: boolean;
  Username: string;
}

export interface RegistryProxyPayload {
  Name: string;
  URL: string;
  TLS: boolean;
  TLSSkipVerify: boolean;
  Authentication: boolean;
  Username: string;
  Password?: string;
}

export interface RegistryCatalog {
  repositories: string[];
}

export interface RegistryTags {
  repository: string;
  tags: string[];
}

export interface RegistryTagPayload {
  repository: string;
  source: string;
  target: string;
}
