import axios, { parseAxiosError } from '@/portainer/services/axios/axios';

import {
  ServiceInstance,
  ServiceInstanceOperation,
  ServiceInstanceTarget,
  ServiceInstanceScheduledBuild,
  CreateServiceInstancePayload,
  UpdateServiceInstancePayload,
  ScheduleServiceInstanceBuildPayload,
} from './types';

const baseUrl = '/service-instances';

export async function getServiceInstances(): Promise<ServiceInstance[]> {
  try {
    const { data } = await axios.get<ServiceInstance[]>(baseUrl);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve service instances');
  }
}

export async function getServiceInstance(id: number): Promise<ServiceInstance> {
  try {
    const { data } = await axios.get<ServiceInstance>(`${baseUrl}/${id}`);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve service instance');
  }
}

export async function createServiceInstance(
  payload: CreateServiceInstancePayload
): Promise<ServiceInstance> {
  try {
    const { data } = await axios.post<ServiceInstance>(baseUrl, payload);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to create service instance');
  }
}

export async function updateServiceInstance(
  id: number,
  payload: UpdateServiceInstancePayload
): Promise<ServiceInstance> {
  try {
    const { data } = await axios.put<ServiceInstance>(
      `${baseUrl}/${id}`,
      payload
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to update service instance');
  }
}

export async function deleteServiceInstance(id: number): Promise<void> {
  try {
    await axios.delete(`${baseUrl}/${id}`);
  } catch (e) {
    throw parseAxiosError(e, 'Unable to delete service instance');
  }
}

export async function getServiceInstanceTargets(
  id: number
): Promise<ServiceInstanceTarget[]> {
  try {
    const { data } = await axios.get<ServiceInstanceTarget[]>(
      `${baseUrl}/${id}/targets`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve service instance targets');
  }
}

export async function getServiceInstanceOperations(
  id: number
): Promise<ServiceInstanceOperation[]> {
  try {
    const { data } = await axios.get<ServiceInstanceOperation[]>(
      `${baseUrl}/${id}/operations`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve service instance operations');
  }
}

export async function getServiceInstanceOperation(
  id: number
): Promise<ServiceInstanceOperation> {
  try {
    const { data } = await axios.get<ServiceInstanceOperation>(
      `/service-instance-operations/${id}`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve service instance operation');
  }
}

export async function deployServiceInstance(
  id: number
): Promise<ServiceInstanceOperation> {
  try {
    const { data } = await axios.post<ServiceInstanceOperation>(
      `${baseUrl}/${id}/deploy`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to deploy service instance');
  }
}

export async function startServiceInstance(
  id: number
): Promise<ServiceInstanceOperation> {
  try {
    const { data } = await axios.post<ServiceInstanceOperation>(
      `${baseUrl}/${id}/start`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to start service instance');
  }
}

export async function stopServiceInstance(
  id: number
): Promise<ServiceInstanceOperation> {
  try {
    const { data } = await axios.post<ServiceInstanceOperation>(
      `${baseUrl}/${id}/stop`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to stop service instance');
  }
}

export async function redeployServiceInstance(
  id: number
): Promise<ServiceInstanceOperation> {
  try {
    const { data } = await axios.post<ServiceInstanceOperation>(
      `${baseUrl}/${id}/redeploy`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to redeploy service instance');
  }
}

export async function refreshServiceInstance(
  id: number
): Promise<ServiceInstance> {
  try {
    const { data } = await axios.post<ServiceInstance>(
      `${baseUrl}/${id}/refresh`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to refresh service instance');
  }
}

export async function scheduleServiceInstanceBuild(
  id: number,
  payload: ScheduleServiceInstanceBuildPayload
): Promise<ServiceInstanceScheduledBuild> {
  try {
    const { data } = await axios.post<ServiceInstanceScheduledBuild>(
      `${baseUrl}/${id}/schedule-build`,
      payload
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to schedule build');
  }
}

export async function getServiceInstanceScheduledBuilds(
  id: number
): Promise<ServiceInstanceScheduledBuild[]> {
  try {
    const { data } = await axios.get<ServiceInstanceScheduledBuild[]>(
      `${baseUrl}/${id}/scheduled-builds`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve scheduled builds');
  }
}

export async function cancelServiceInstanceScheduledBuild(
  id: number
): Promise<void> {
  try {
    await axios.delete(`/service-instance-scheduled-builds/${id}`);
  } catch (e) {
    throw parseAxiosError(e, 'Unable to cancel scheduled build');
  }
}
