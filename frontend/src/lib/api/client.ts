/**
 * Runtime-aware API client using native fetch.
 * All endpoints use POST with JSON body (Huma RPC convention).
 *
 * Auth handling: on 401 response, redirects browser to /login with
 * the current path preserved as a redirect target. After successful
 * login, the user is sent back to the original path.
 */

import { getApiUrl, getRuntimeConfig } from '../runtime';
import type {
  PersistedConfig,
  UpdateConfigRequest,
  Device,
  ListDevicesRequest,
  ListDevicesResponse,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  Source,
  ListSourcesRequest,
  ListSourcesResponse,
  CreateSourceRequest,
  UpdateSourceRequest,
  SourceType,
  ListSourceTypesResponse,
  SourceSchedule,
  ListSchedulesRequest,
  ListSchedulesResponse,
  CreateScheduleRequest,
  UpdateScheduleRequest,
  DeviceSubscription,
  ListSubscriptionsRequest,
  ListSubscriptionsResponse,
  CreateSubscriptionRequest,
  UpdateSubscriptionRequest,
  Image,
  ListImagesRequest,
  ListImagesResponse,
  ListDeviceImagesRequest,
  SetImageFavoriteRequest,
  BlacklistImageRequest,
  DeleteImageRequest,
  Job,
  ListJobsRequest,
  ListJobsResponse,
  GetJobRequest,
  RuntimeStatus,
} from './types';

const apiUrl = () => getApiUrl();
const { basePath } = getRuntimeConfig();

/** Store the path to redirect to after login (session-scoped). */
let redirectAfterLogin = '/';

/**
 * Set the path to redirect to after a successful login.
 * Call this before navigating to /login.
 */
export function setRedirectAfterLogin(path: string) {
  redirectAfterLogin = path || '/';
}

/**
 * Get the stored redirect path and reset it.
 */
export function popRedirectAfterLogin(): string {
  const r = redirectAfterLogin;
  redirectAfterLogin = '/';
  return r;
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const url = `${apiUrl()}${path}`;
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : JSON.stringify({}),
  });

  // 204 No Content – nothing to parse
  if (res.status === 204) {
    return undefined as T;
  }

  if (!res.ok) {
    if (res.status === 401) {
      // Preserve current location before redirecting to login
      const target = window.location.pathname.replace(basePath, '') || '/';
      const loginUrl = `${basePath}/login${target !== '/' ? '?redirect=' + encodeURIComponent(target) : ''}`;
      window.location.href = loginUrl;
      throw new Error('Unauthorized – redirecting to login');
    }
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || `HTTP ${res.status}`);
  }

  const data = await res.json();
  return (data.body ?? data) as T;
}

// ==================== Config (Persisted) ====================

export async function getConfig(): Promise<PersistedConfig> {
  return post<PersistedConfig>('/api/v1/configs/GetConfig', {});
}

export async function updateConfig(body: UpdateConfigRequest): Promise<PersistedConfig> {
  return post<PersistedConfig>('/api/v1/configs/UpdateConfig', body);
}

// ==================== Devices ====================

export async function listDevices(body?: ListDevicesRequest): Promise<ListDevicesResponse> {
  return post('/api/v1/devices/ListDevices', body ?? {});
}

export async function getDevice(id: string): Promise<Device> {
  return post<Device>('/api/v1/devices/GetDevice', { id });
}

export async function createDevice(body: CreateDeviceRequest): Promise<Device> {
  return post<Device>('/api/v1/devices/CreateDevice', body);
}

export async function updateDevice(body: UpdateDeviceRequest): Promise<Device> {
  return post<Device>('/api/v1/devices/UpdateDevice', body);
}

export async function deleteDevice(id: string): Promise<void> {
  await post('/api/v1/devices/DeleteDevice', { id });
}

// ==================== Sources ====================

export async function listSources(body?: ListSourcesRequest): Promise<ListSourcesResponse> {
  return post('/api/v1/sources/ListSources', body ?? {});
}

export async function getSource(id: string): Promise<Source> {
  return post<Source>('/api/v1/sources/GetSource', { id });
}

export async function createSource(body: CreateSourceRequest): Promise<Source> {
  return post<Source>('/api/v1/sources/CreateSource', body);
}

export async function updateSource(body: UpdateSourceRequest): Promise<Source> {
  return post<Source>('/api/v1/sources/UpdateSource', body);
}

export async function deleteSource(id: string): Promise<void> {
  await post('/api/v1/sources/DeleteSource', { id });
}

export async function listSourceTypes(): Promise<ListSourceTypesResponse> {
  return post('/api/v1/source_types/ListSourceTypes', {});
}

// ==================== Source Schedules ====================

export async function listSchedules(body?: ListSchedulesRequest): Promise<ListSchedulesResponse> {
  return post('/api/v1/source_schedules/ListSourceSchedules', body ?? {});
}

export async function getSchedule(id: string): Promise<SourceSchedule> {
  return post<SourceSchedule>('/api/v1/source_schedules/GetSourceSchedule', { id });
}

export async function createSchedule(body: CreateScheduleRequest): Promise<SourceSchedule> {
  return post<SourceSchedule>('/api/v1/source_schedules/CreateSourceSchedule', body);
}

export async function updateSchedule(body: UpdateScheduleRequest): Promise<SourceSchedule> {
  return post<SourceSchedule>('/api/v1/source_schedules/UpdateSourceSchedule', body);
}

export async function deleteSchedule(id: string): Promise<void> {
  await post('/api/v1/source_schedules/DeleteSourceSchedule', { id });
}

// ==================== Device Subscriptions ====================

export async function listSubscriptions(body?: ListSubscriptionsRequest): Promise<ListSubscriptionsResponse> {
  return post('/api/v1/device_subscriptions/ListDeviceSubscriptions', body ?? {});
}

export async function createSubscription(body: CreateSubscriptionRequest): Promise<DeviceSubscription> {
  return post<DeviceSubscription>('/api/v1/device_subscriptions/CreateDeviceSubscription', body);
}

export async function updateSubscription(body: UpdateSubscriptionRequest): Promise<DeviceSubscription> {
  return post<DeviceSubscription>('/api/v1/device_subscriptions/UpdateDeviceSubscription', body);
}

export async function deleteSubscription(id: string): Promise<void> {
  await post('/api/v1/device_subscriptions/DeleteDeviceSubscription', { id });
}

// ==================== Images ====================

export async function listImages(body?: ListImagesRequest): Promise<ListImagesResponse> {
  return post('/api/v1/images/ListImages', body ?? {});
}

export async function listDeviceImages(body: ListDeviceImagesRequest): Promise<ListImagesResponse> {
  return post('/api/v1/images/ListDeviceImages', body);
}

export async function getImage(id: string): Promise<Image> {
  return post<Image>('/api/v1/images/GetImage', { id });
}

export async function setImageFavorite(body: SetImageFavoriteRequest): Promise<void> {
  await post('/api/v1/images/SetImageFavorite', body);
}

export async function blacklistImage(body: BlacklistImageRequest): Promise<void> {
  await post('/api/v1/images/BlacklistImage', body);
}

export async function deleteImage(body: DeleteImageRequest): Promise<void> {
  await post('/api/v1/images/DeleteImage', body);
}

// ==================== Jobs ====================

export async function listJobs(body?: ListJobsRequest): Promise<ListJobsResponse> {
  return post('/api/v1/jobs/ListJobs', body ?? {});
}

export async function getJob(body: GetJobRequest): Promise<Job> {
  return post<Job>('/api/v1/jobs/GetJob', body);
}

// ==================== Runtime Status ====================

export async function getRuntimeStatus(): Promise<RuntimeStatus> {
  return post('/api/v1/runtime_status/GetRuntimeStatus', {});
}

// ==================== Auth ====================

export async function login(username: string, password: string): Promise<void> {
  const loginUrl = `${apiUrl()}/api/login`;
  const res = await fetch(loginUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    if (res.status === 401) {
      throw new Error('Invalid username or password');
    }
    throw new Error(`Login failed: ${res.status}`);
  }
}

export async function logout(): Promise<void> {
  const logoutUrl = `${apiUrl()}/api/logout`;
  await fetch(logoutUrl, { method: 'POST' });
}
