import {
  ServiceInstance,
  ServiceInstanceOperation,
  ServiceInstanceOperationStatuses,
  ServiceInstanceOperationTypes,
  ServiceInstanceScheduledBuild,
  ServiceInstanceScheduledBuildStatuses,
  ServiceInstanceScheduledBuildTargetStatuses,
  ServiceInstanceTarget,
  ServiceInstanceTargetStatuses,
  ServiceInstanceTargetTypes,
} from '../types';

export const mockServiceInstance: ServiceInstance = {
  Id: 1,
  Name: 'production-web',
  Description: 'production web service',
  TargetType: ServiceInstanceTargetTypes.GROUP,
  GroupId: 1,
  EnvironmentIds: [],
  StackName: 'si-1-production-web',
  ComposeFile: 'services:\n  web:\n    image: nginx:latest',
  Status: 2,
  CreatedBy: 'admin',
  CreatedAt: 1700000000,
  UpdatedAt: 1700000000,
};

export const mockServiceInstanceOperation: ServiceInstanceOperation = {
  Id: 1,
  ServiceInstanceId: 1,
  Type: ServiceInstanceOperationTypes.DEPLOY,
  Status: ServiceInstanceOperationStatuses.SUCCESS,
  UserId: 1,
  StartedAt: 1700000000,
  FinishedAt: 1700000010,
  Results: [
    {
      EnvironmentId: 1,
      Status: ServiceInstanceTargetStatuses.SUCCESS,
    },
    {
      EnvironmentId: 2,
      Status: ServiceInstanceTargetStatuses.SUCCESS,
    },
  ],
};

export const mockServiceInstanceScheduledBuild: ServiceInstanceScheduledBuild =
  {
    Id: 1,
    ServiceInstanceId: 1,
    ComposeFile: 'services:\n  web:\n    image: nginx:latest',
    DeployAt: 1700003600,
    Status: ServiceInstanceScheduledBuildStatuses.PENDING,
    UserId: 1,
    CreatedAt: 1700000000,
    Results: [
      {
        EnvironmentId: 1,
        Status: ServiceInstanceScheduledBuildTargetStatuses.PENDING,
      },
    ],
  };

export const mockServiceInstanceTargets: ServiceInstanceTarget[] = [
  {
    EnvironmentId: 1,
    Environment: { Id: 1, Name: 'prod-a' },
    Stack: { Id: 1, Name: 'si-1-production-web', Status: 1 },
    Missing: false,
  },
  {
    EnvironmentId: 2,
    Environment: { Id: 2, Name: 'prod-b' },
    Stack: { Id: 2, Name: 'si-1-production-web', Status: 1 },
    Missing: false,
  },
];
