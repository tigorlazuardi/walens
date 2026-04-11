/**
 * API client using openapi-fetch.
 *
 * All RPC endpoints use POST with JSON body (Huma convention).
 * Auth handling: on 401 response, redirects browser to /login with
 * the current path preserved as a redirect target. After successful
 * login, the user is sent back to the original path.
 */

import createClient, { type Client } from 'openapi-fetch';
import { getRuntimeConfig } from '../runtime';
import type { paths } from './types';
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

const { basePath } = getRuntimeConfig();

/**
 * Create a new openapi-fetch client configured for the Walens API.
 * The baseUrl determines the API root; paths are relative to it.
 *
 * Note: openapi-fetch paths already include /api/..., so we use
 * basePath (not apiBase) as the base URL. For root deployment
 * basePath is '/'; for subpath deployment it is '/subpath'.
 */
export function createApiClient(baseUrl: string): Client<paths> {
  return createClient<paths>({
    baseUrl,
    fetchOptions: () => ({
      credentials: 'include',
    }),
  });
}

/** Default API client instance. */
export const apiClient = createApiClient(basePath);

// ==================== Auth Helpers ====================

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

// ==================== Internal 401 Handler ====================

/** Handles 401 by redirecting browser to login page. */
function handleAuthError() {
  const target = window.location.pathname.replace(basePath, '') || '/';
  const loginUrl = `${basePath}/login${target !== '/' ? '?redirect=' + encodeURIComponent(target) : ''}`;
  window.location.href = loginUrl;
  throw new Error('Unauthorized – redirecting to login');
}

// ==================== Config (Persisted) ====================

export async function getConfig(): Promise<PersistedConfig> {
  const res = await apiClient.POST('/api/v1/configs/GetConfig', { body: {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function updateConfig(body: UpdateConfigRequest): Promise<PersistedConfig> {
  const res = await apiClient.POST('/api/v1/configs/UpdateConfig', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

// ==================== Devices ====================

export async function listDevices(body?: ListDevicesRequest): Promise<ListDevicesResponse> {
  const res = await apiClient.POST('/api/v1/devices/ListDevices', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function getDevice(id: string): Promise<Device> {
  const res = await apiClient.POST('/api/v1/devices/GetDevice', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function createDevice(body: CreateDeviceRequest): Promise<Device> {
  const res = await apiClient.POST('/api/v1/devices/CreateDevice', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function updateDevice(body: UpdateDeviceRequest): Promise<Device> {
  const res = await apiClient.POST('/api/v1/devices/UpdateDevice', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function deleteDevice(id: string): Promise<void> {
  const res = await apiClient.POST('/api/v1/devices/DeleteDevice', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

// ==================== Sources ====================

export async function listSources(body?: ListSourcesRequest): Promise<ListSourcesResponse> {
  const res = await apiClient.POST('/api/v1/sources/ListSources', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function getSource(id: string): Promise<Source> {
  const res = await apiClient.POST('/api/v1/sources/GetSource', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function createSource(body: CreateSourceRequest): Promise<Source> {
  const res = await apiClient.POST('/api/v1/sources/CreateSource', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function updateSource(body: UpdateSourceRequest): Promise<Source> {
  const res = await apiClient.POST('/api/v1/sources/UpdateSource', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function deleteSource(id: string): Promise<void> {
  const res = await apiClient.POST('/api/v1/sources/DeleteSource', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

export async function listSourceTypes(): Promise<ListSourceTypesResponse> {
  const res = await apiClient.POST('/api/v1/source_types/ListSourceTypes', { body: {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

// ==================== Source Schedules ====================

export async function listSchedules(body?: ListSchedulesRequest): Promise<ListSchedulesResponse> {
  const res = await apiClient.POST('/api/v1/source_schedules/ListSourceSchedules', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function getSchedule(id: string): Promise<SourceSchedule> {
  const res = await apiClient.POST('/api/v1/source_schedules/GetSourceSchedule', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function createSchedule(body: CreateScheduleRequest): Promise<SourceSchedule> {
  const res = await apiClient.POST('/api/v1/source_schedules/CreateSourceSchedule', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  // Response body wraps with { schedule, warnings }
  return res.data.schedule ?? res.data as unknown as SourceSchedule;
}

export async function updateSchedule(body: UpdateScheduleRequest): Promise<SourceSchedule> {
  const res = await apiClient.POST('/api/v1/source_schedules/UpdateSourceSchedule', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data.schedule ?? res.data as unknown as SourceSchedule;
}

export async function deleteSchedule(id: string): Promise<void> {
  const res = await apiClient.POST('/api/v1/source_schedules/DeleteSourceSchedule', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

// ==================== Device Subscriptions ====================

export async function listSubscriptions(body?: ListSubscriptionsRequest): Promise<ListSubscriptionsResponse> {
  const res = await apiClient.POST('/api/v1/device_subscriptions/ListDeviceSubscriptions', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function createSubscription(body: CreateSubscriptionRequest): Promise<DeviceSubscription> {
  const res = await apiClient.POST('/api/v1/device_subscriptions/CreateDeviceSubscription', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function updateSubscription(body: UpdateSubscriptionRequest): Promise<DeviceSubscription> {
  const res = await apiClient.POST('/api/v1/device_subscriptions/UpdateDeviceSubscription', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function deleteSubscription(id: string): Promise<void> {
  const res = await apiClient.POST('/api/v1/device_subscriptions/DeleteDeviceSubscription', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

// ==================== Images ====================

export async function listImages(body?: ListImagesRequest): Promise<ListImagesResponse> {
  const res = await apiClient.POST('/api/v1/images/ListImages', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function listDeviceImages(body: ListDeviceImagesRequest): Promise<ListImagesResponse> {
  const res = await apiClient.POST('/api/v1/images/ListDeviceImages', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function getImage(id: string): Promise<Image> {
  const res = await apiClient.POST('/api/v1/images/GetImage', { body: { id } });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

// NOTE: schema uses `is_favorite` but the app uses `favorite` in SetImageFavoriteRequest
export async function setImageFavorite(body: SetImageFavoriteRequest & { favorite?: boolean }): Promise<void> {
  const res = await apiClient.POST('/api/v1/images/SetImageFavorite', {
    body: { id: body.id, is_favorite: body.is_favorite ?? body.favorite },
  });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

// NOTE: schema uses `image_id` but the app uses `id` in BlacklistImageRequest
export async function blacklistImage(body: BlacklistImageRequest & { id?: string }): Promise<void> {
  const res = await apiClient.POST('/api/v1/images/BlacklistImage', {
    body: { image_id: body.image_id ?? body.id, reason: body.reason },
  });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

export async function deleteImage(body: DeleteImageRequest): Promise<void> {
  const res = await apiClient.POST('/api/v1/images/DeleteImage', { body });
  if (res.response.status === 401) handleAuthError();
  if (res.response.status === 204 || res.response.status === 200) return;
  if (!res.data) throw new Error('No data received');
}

// ==================== Jobs ====================

export async function listJobs(body?: ListJobsRequest): Promise<ListJobsResponse> {
  const res = await apiClient.POST('/api/v1/jobs/ListJobs', { body: body ?? {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

export async function getJob(body: GetJobRequest): Promise<Job> {
  const res = await apiClient.POST('/api/v1/jobs/GetJob', { body });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

// ==================== Runtime Status ====================

export async function getRuntimeStatus(): Promise<RuntimeStatus> {
  const res = await apiClient.POST('/api/v1/runtime_status/GetRuntimeStatus', { body: {} });
  if (res.response.status === 401) handleAuthError();
  if (!res.data) throw new Error('No data received');
  return res.data;
}

// ==================== Auth ====================

export async function login(username: string, password: string): Promise<void> {
  const loginUrl = `${basePath}/api/login`;
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
  const logoutUrl = `${basePath}/api/logout`;
  await fetch(logoutUrl, { method: 'POST' });
}
