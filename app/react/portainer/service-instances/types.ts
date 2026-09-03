import {
  EnvironmentId,
  EnvironmentGroupId,
} from '@/react/portainer/environments/types';

export type ServiceInstanceId = number;

export type ServiceInstanceOperationId = number;

export type ServiceInstanceTargetType =
  | typeof ServiceInstanceTargetTypes.GROUP
  | typeof ServiceInstanceTargetTypes.ENVIRONMENTS;

export const ServiceInstanceTargetTypes = {
  GROUP: 1,
  ENVIRONMENTS: 2,
} as const;

export type ServiceInstanceStatus =
  | typeof ServiceInstanceStatuses.UNKNOWN
  | typeof ServiceInstanceStatuses.DEPLOYING
  | typeof ServiceInstanceStatuses.RUNNING
  | typeof ServiceInstanceStatuses.STOPPED
  | typeof ServiceInstanceStatuses.PARTIAL
  | typeof ServiceInstanceStatuses.FAILED;

export const ServiceInstanceStatuses = {
  UNKNOWN: 0,
  DEPLOYING: 1,
  RUNNING: 2,
  STOPPED: 3,
  PARTIAL: 4,
  FAILED: 5,
} as const;

export type ServiceInstanceOperationType =
  | typeof ServiceInstanceOperationTypes.DEPLOY
  | typeof ServiceInstanceOperationTypes.START
  | typeof ServiceInstanceOperationTypes.STOP
  | typeof ServiceInstanceOperationTypes.REDEPLOY
  | typeof ServiceInstanceOperationTypes.REFRESH;

export const ServiceInstanceOperationTypes = {
  DEPLOY: 1,
  START: 2,
  STOP: 3,
  REDEPLOY: 4,
  REFRESH: 5,
} as const;

export type ServiceInstanceOperationStatus =
  | typeof ServiceInstanceOperationStatuses.PENDING
  | typeof ServiceInstanceOperationStatuses.RUNNING
  | typeof ServiceInstanceOperationStatuses.SUCCESS
  | typeof ServiceInstanceOperationStatuses.PARTIAL_SUCCESS
  | typeof ServiceInstanceOperationStatuses.FAILED
  | typeof ServiceInstanceOperationStatuses.CANCELLED;

export const ServiceInstanceOperationStatuses = {
  PENDING: 1,
  RUNNING: 2,
  SUCCESS: 3,
  PARTIAL_SUCCESS: 4,
  FAILED: 5,
  CANCELLED: 6,
} as const;

export type ServiceInstanceTargetStatus =
  | typeof ServiceInstanceTargetStatuses.PENDING
  | typeof ServiceInstanceTargetStatuses.RUNNING
  | typeof ServiceInstanceTargetStatuses.SUCCESS
  | typeof ServiceInstanceTargetStatuses.FAILED
  | typeof ServiceInstanceTargetStatuses.SKIPPED;

export const ServiceInstanceTargetStatuses = {
  PENDING: 1,
  RUNNING: 2,
  SUCCESS: 3,
  FAILED: 4,
  SKIPPED: 5,
} as const;

export interface ServiceInstance {
  Id: ServiceInstanceId;
  Name: string;
  Description: string;
  TargetType: ServiceInstanceTargetType;
  GroupId?: EnvironmentGroupId;
  EnvironmentIds?: EnvironmentId[];
  StackName: string;
  ComposeFile: string;
  Env?: Array<{ name: string; value: string }>;
  Status: ServiceInstanceStatus;
  CreatedBy: string;
  CreatedAt: number;
  UpdatedAt: number;
}

export interface ServiceInstanceTargetResult {
  EnvironmentId: EnvironmentId;
  Status: ServiceInstanceTargetStatus;
  Error?: string;
}

export interface ServiceInstanceOperation {
  Id: ServiceInstanceOperationId;
  ServiceInstanceId: ServiceInstanceId;
  Type: ServiceInstanceOperationType;
  Status: ServiceInstanceOperationStatus;
  UserId: number;
  StartedAt: number;
  FinishedAt?: number;
  Results: ServiceInstanceTargetResult[];
}

export interface ServiceInstanceTarget {
  EnvironmentId: EnvironmentId;
  Environment?: {
    Id: EnvironmentId;
    Name: string;
  };
  Stack?: {
    Id: number;
    Name: string;
    Status: number;
  };
  Missing: boolean;
}

export interface CreateServiceInstancePayload {
  Name: string;
  Description: string;
  TargetType: ServiceInstanceTargetType;
  GroupId?: EnvironmentGroupId;
  EnvironmentIds?: EnvironmentId[];
  ComposeFile: string;
  Env?: Array<{ name: string; value: string }>;
}

export interface UpdateServiceInstancePayload {
  Name?: string;
  Description?: string;
  TargetType?: ServiceInstanceTargetType;
  GroupId?: EnvironmentGroupId;
  EnvironmentIds?: EnvironmentId[];
  ComposeFile?: string;
  Env?: Array<{ name: string; value: string }>;
}
