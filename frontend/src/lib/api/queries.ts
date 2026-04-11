/**
 * TanStack Query hooks for Svelte using @tanstack/svelte-query v6.
 * Query keys must use serializable values (no function refs).
 */

import { createQuery, createMutation } from '@tanstack/svelte-query';
import { QueryClient } from '../query-client';
import * as api from './client';
import type {
  ListDevicesRequest,
  ListSourcesRequest,
  ListSchedulesRequest,
  ListSubscriptionsRequest,
  ListImagesRequest,
  ListJobsRequest,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  CreateSourceRequest,
  UpdateSourceRequest,
  CreateScheduleRequest,
  UpdateScheduleRequest,
  CreateSubscriptionRequest,
  UpdateSubscriptionRequest,
  SetImageFavoriteRequest,
  BlacklistImageRequest,
  DeleteImageRequest,
  UpdateConfigRequest,
} from './types';

// ==================== Config ====================

export function useConfig() {
  return createQuery({
    queryKey: ['config'],
    queryFn: () => api.getConfig(),
  });
}

export function useUpdateConfig() {
  return createMutation({
    mutationFn: (body: UpdateConfigRequest) => api.updateConfig(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['config'] }),
  });
}

// ==================== Devices ====================

export function useDevices(input: ListDevicesRequest = {}) {
  return createQuery({
    queryKey: ['devices', input],
    queryFn: () => api.listDevices(input),
  });
}

export function useDevice(id: string | undefined) {
  return createQuery({
    queryKey: ['device', id ?? ''],
    queryFn: () => (id ? api.getDevice(id) : Promise.resolve(null)),
    enabled: !!id,
  });
}

export function useCreateDevice() {
  return createMutation({
    mutationFn: (body: CreateDeviceRequest) => api.createDevice(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['devices'] }),
  });
}

export function useUpdateDevice() {
  return createMutation({
    mutationFn: (body: UpdateDeviceRequest) => api.updateDevice(body),
    onSuccess: () => {
      QueryClient.invalidateQueries({ queryKey: ['devices'] });
      if (body.id) {
        QueryClient.invalidateQueries({ queryKey: ['device', body.id] });
      }
    },
  });
}

export function useDeleteDevice() {
  return createMutation({
    mutationFn: (id: string) => api.deleteDevice(id),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['devices'] }),
  });
}

// ==================== Sources ====================

export function useSources(input: ListSourcesRequest = {}) {
  return createQuery({
    queryKey: ['sources', input],
    queryFn: () => api.listSources(input),
  });
}

export function useSource(id: string | undefined) {
  return createQuery({
    queryKey: ['source', id ?? ''],
    queryFn: () => (id ? api.getSource(id) : Promise.resolve(null)),
    enabled: !!id,
  });
}

export function useSourceTypes() {
  return createQuery({
    queryKey: ['sourceTypes'],
    queryFn: () => api.listSourceTypes(),
  });
}

export function useCreateSource() {
  return createMutation({
    mutationFn: (body: CreateSourceRequest) => api.createSource(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['sources'] }),
  });
}

export function useUpdateSource() {
  return createMutation({
    mutationFn: (body: UpdateSourceRequest) => api.updateSource(body),
    onSuccess: () => {
      QueryClient.invalidateQueries({ queryKey: ['sources'] });
      if (body.id) {
        QueryClient.invalidateQueries({ queryKey: ['source', body.id] });
      }
    },
  });
}

export function useDeleteSource() {
  return createMutation({
    mutationFn: (id: string) => api.deleteSource(id),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['sources'] }),
  });
}

// ==================== Source Schedules ====================

export function useSchedules(input: ListSchedulesRequest = {}) {
  return createQuery({
    queryKey: ['schedules', input],
    queryFn: () => api.listSchedules(input),
  });
}

export function useSchedule(id: string | undefined) {
  return createQuery({
    queryKey: ['schedule', id ?? ''],
    queryFn: () => (id ? api.getSchedule(id) : Promise.resolve(null)),
    enabled: !!id,
  });
}

export function useCreateSchedule() {
  return createMutation({
    mutationFn: (body: CreateScheduleRequest) => api.createSchedule(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['schedules'] }),
  });
}

export function useUpdateSchedule() {
  return createMutation({
    mutationFn: (body: UpdateScheduleRequest) => api.updateSchedule(body),
    onSuccess: () => {
      QueryClient.invalidateQueries({ queryKey: ['schedules'] });
      if (body.id) {
        QueryClient.invalidateQueries({ queryKey: ['schedule', body.id] });
      }
    },
  });
}

export function useDeleteSchedule() {
  return createMutation({
    mutationFn: (id: string) => api.deleteSchedule(id),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['schedules'] }),
  });
}

// ==================== Device Subscriptions ====================

export function useSubscriptions(input: ListSubscriptionsRequest = {}) {
  return createQuery({
    queryKey: ['subscriptions', input],
    queryFn: () => api.listSubscriptions(input),
  });
}

export function useCreateSubscription() {
  return createMutation({
    mutationFn: (body: CreateSubscriptionRequest) => api.createSubscription(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['subscriptions'] }),
  });
}

export function useUpdateSubscription() {
  return createMutation({
    mutationFn: (body: UpdateSubscriptionRequest) => api.updateSubscription(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['subscriptions'] }),
  });
}

export function useDeleteSubscription() {
  return createMutation({
    mutationFn: (id: string) => api.deleteSubscription(id),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['subscriptions'] }),
  });
}

// ==================== Images ====================

export function useImages(input: ListImagesRequest = {}) {
  return createQuery({
    queryKey: ['images', input],
    queryFn: () => api.listImages(input),
  });
}

export function useDeviceImages(deviceId: string | undefined) {
  return createQuery({
    queryKey: ['deviceImages', deviceId ?? ''],
    queryFn: () =>
      deviceId
        ? api.listDeviceImages({ device_id: deviceId })
        : Promise.resolve({ items: [], total: 0 }),
    enabled: !!deviceId,
  });
}

export function useSetImageFavorite() {
  return createMutation({
    mutationFn: (body: SetImageFavoriteRequest) => api.setImageFavorite(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['images'] }),
  });
}

export function useBlacklistImage() {
  return createMutation({
    mutationFn: (body: BlacklistImageRequest) => api.blacklistImage(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['images'] }),
  });
}

export function useDeleteImage() {
  return createMutation({
    mutationFn: (body: DeleteImageRequest) => api.deleteImage(body),
    onSuccess: () => QueryClient.invalidateQueries({ queryKey: ['images'] }),
  });
}

// ==================== Jobs ====================

export function useJobs(input: ListJobsRequest = {}) {
  return createQuery({
    queryKey: ['jobs', input],
    queryFn: () => api.listJobs(input),
  });
}

export function useJob(id: string | undefined) {
  return createQuery({
    queryKey: ['job', id ?? ''],
    queryFn: () => (id ? api.getJob({ id }) : Promise.resolve(null)),
    enabled: !!id,
  });
}

// ==================== Runtime Status ====================

export function useRuntimeStatus() {
  return createQuery({
    queryKey: ['runtimeStatus'],
    queryFn: () => api.getRuntimeStatus(),
    refetchInterval: 5000,
  });
}
