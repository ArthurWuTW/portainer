import axios, { parseAxiosError } from '@/portainer/services/axios/axios';

import {
  RegistryCatalog,
  RegistryProxy,
  RegistryProxyId,
  RegistryProxyPayload,
  RegistryTagPayload,
  RegistryTags,
} from './types';

const baseUrl = '/registry-proxies';

export async function getRegistryProxies(): Promise<RegistryProxy[]> {
  try {
    const { data } = await axios.get<RegistryProxy[]>(baseUrl);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve registry proxies');
  }
}

export async function getRegistryProxy(
  id: RegistryProxyId
): Promise<RegistryProxy> {
  try {
    const { data } = await axios.get<RegistryProxy>(`${baseUrl}/${id}`);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve registry proxy');
  }
}

export async function createRegistryProxy(
  payload: RegistryProxyPayload
): Promise<RegistryProxy> {
  try {
    const { data } = await axios.post<RegistryProxy>(baseUrl, payload);
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to create registry proxy');
  }
}

export async function updateRegistryProxy(
  id: RegistryProxyId,
  payload: RegistryProxyPayload
): Promise<RegistryProxy> {
  try {
    const { data } = await axios.put<RegistryProxy>(
      `${baseUrl}/${id}`,
      payload
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to update registry proxy');
  }
}

export async function deleteRegistryProxy(id: RegistryProxyId): Promise<void> {
  try {
    await axios.delete(`${baseUrl}/${id}`);
  } catch (e) {
    throw parseAxiosError(e, 'Unable to delete registry proxy');
  }
}

export async function getRegistryCatalog(
  id: RegistryProxyId
): Promise<RegistryCatalog> {
  try {
    const { data } = await axios.get<RegistryCatalog>(
      `${baseUrl}/${id}/catalog`
    );
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve registry catalog');
  }
}

export async function getRegistryTags(
  id: RegistryProxyId,
  repository: string
): Promise<RegistryTags> {
  try {
    const { data } = await axios.get<RegistryTags>(`${baseUrl}/${id}/tags`, {
      params: { repository },
    });
    return data;
  } catch (e) {
    throw parseAxiosError(e, 'Unable to retrieve image versions');
  }
}

export async function deleteRegistryImage(
  id: RegistryProxyId,
  repository: string,
  reference: string
): Promise<void> {
  try {
    await axios.delete(`${baseUrl}/${id}/manifest`, {
      params: { repository, reference },
    });
  } catch (e) {
    throw parseAxiosError(e, 'Unable to delete image');
  }
}

export async function addRegistryTag(
  id: RegistryProxyId,
  payload: RegistryTagPayload
): Promise<void> {
  try {
    await axios.post(`${baseUrl}/${id}/tags`, payload);
  } catch (e) {
    throw parseAxiosError(e, 'Unable to tag image');
  }
}
