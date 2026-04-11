/**
 * TanStack Query hooks for Svelte using @tanstack/svelte-query v6.
 * Query keys must use serializable values (no function refs).
 */

import { createQuery, createMutation } from "@tanstack/svelte-query";
import { QueryClient } from "../query-client";
import * as api from "./client";
import { apiRoutes } from "./routes";
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
} from "./types";

function resolveInput<T>(input: T | (() => T)): T {
    return typeof input === "function" ? (input as () => T)() : input;
}

// ==================== Config ====================

export function useConfig() {
    return createQuery(() => ({
        queryKey: ["config"],
        queryFn: () => api.request(apiRoutes.configs.get, {}),
    }));
}

export function useUpdateConfig() {
    return createMutation(() => ({
        mutationFn: (body: UpdateConfigRequest) => api.request(apiRoutes.configs.update, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["config"] }),
    }));
}

// ==================== Devices ====================

export function useDevices(input: ListDevicesRequest | (() => ListDevicesRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["devices", resolved],
            queryFn: () => api.request(apiRoutes.devices.list, resolved),
        };
    });
}

export function useDevice(id: string | undefined) {
    return createQuery(() => ({
        queryKey: ["device", id ?? ""],
        queryFn: () => (id ? api.request(apiRoutes.devices.get, { id }) : Promise.resolve(null)),
        enabled: !!id,
    }));
}

export function useCreateDevice() {
    return createMutation(() => ({
        mutationFn: (body: CreateDeviceRequest) => api.request(apiRoutes.devices.create, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["devices"] }),
    }));
}

export function useUpdateDevice() {
    return createMutation(() => ({
        mutationFn: (body: UpdateDeviceRequest) => api.request(apiRoutes.devices.update, body),
        onSuccess: (_data, body) => {
            QueryClient.invalidateQueries({ queryKey: ["devices"] });
            if (body.id) {
                QueryClient.invalidateQueries({ queryKey: ["device", body.id] });
            }
        },
    }));
}

export function useDeleteDevice() {
    return createMutation(() => ({
        mutationFn: (id: string) => api.request(apiRoutes.devices.delete, { id }),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["devices"] }),
    }));
}

// ==================== Sources ====================

export function useSources(input: ListSourcesRequest | (() => ListSourcesRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["sources", resolved],
            queryFn: () => api.request(apiRoutes.sources.list, resolved),
        };
    });
}

export function useSource(id: string | undefined) {
    return createQuery(() => ({
        queryKey: ["source", id ?? ""],
        queryFn: () => (id ? api.request(apiRoutes.sources.get, { id }) : Promise.resolve(null)),
        enabled: !!id,
    }));
}

export function useSourceTypes() {
    return createQuery(() => ({
        queryKey: ["sourceTypes"],
        queryFn: () => api.request(apiRoutes.sourceTypes.list, {}),
    }));
}

export function useCreateSource() {
    return createMutation(() => ({
        mutationFn: (body: CreateSourceRequest) => api.request(apiRoutes.sources.create, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["sources"] }),
    }));
}

export function useUpdateSource() {
    return createMutation(() => ({
        mutationFn: (body: UpdateSourceRequest) => api.request(apiRoutes.sources.update, body),
        onSuccess: (_data, body) => {
            QueryClient.invalidateQueries({ queryKey: ["sources"] });
            if (body.id) {
                QueryClient.invalidateQueries({ queryKey: ["source", body.id] });
            }
        },
    }));
}

export function useDeleteSource() {
    return createMutation(() => ({
        mutationFn: (id: string) => api.request(apiRoutes.sources.delete, { id }),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["sources"] }),
    }));
}

// ==================== Source Schedules ====================

export function useSchedules(input: ListSchedulesRequest | (() => ListSchedulesRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["schedules", resolved],
            queryFn: () => api.request(apiRoutes.schedules.list, resolved),
        };
    });
}

export function useSchedule(id: string | undefined) {
    return createQuery(() => ({
        queryKey: ["schedule", id ?? ""],
        queryFn: () => (id ? api.request(apiRoutes.schedules.get, { id }) : Promise.resolve(null)),
        enabled: !!id,
    }));
}

export function useCreateSchedule() {
    return createMutation(() => ({
        mutationFn: (body: CreateScheduleRequest) => api.request(apiRoutes.schedules.create, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["schedules"] }),
    }));
}

export function useUpdateSchedule() {
    return createMutation(() => ({
        mutationFn: (body: UpdateScheduleRequest) => api.request(apiRoutes.schedules.update, body),
        onSuccess: (_data, body) => {
            QueryClient.invalidateQueries({ queryKey: ["schedules"] });
            if (body.id) {
                QueryClient.invalidateQueries({ queryKey: ["schedule", body.id] });
            }
        },
    }));
}

export function useDeleteSchedule() {
    return createMutation(() => ({
        mutationFn: (id: string) => api.request(apiRoutes.schedules.delete, { id }),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["schedules"] }),
    }));
}

// ==================== Device Subscriptions ====================

export function useSubscriptions(input: ListSubscriptionsRequest | (() => ListSubscriptionsRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["subscriptions", resolved],
            queryFn: () => api.request(apiRoutes.subscriptions.list, resolved),
        };
    });
}

export function useCreateSubscription() {
    return createMutation(() => ({
        mutationFn: (body: CreateSubscriptionRequest) => api.request(apiRoutes.subscriptions.create, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["subscriptions"] }),
    }));
}

export function useUpdateSubscription() {
    return createMutation(() => ({
        mutationFn: (body: UpdateSubscriptionRequest) => api.request(apiRoutes.subscriptions.update, body),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["subscriptions"] }),
    }));
}

export function useDeleteSubscription() {
    return createMutation(() => ({
        mutationFn: (id: string) => api.request(apiRoutes.subscriptions.delete, { id }),
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["subscriptions"] }),
    }));
}

// ==================== Images ====================

export function useImages(input: ListImagesRequest | (() => ListImagesRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["images", resolved],
            queryFn: () => api.request(apiRoutes.images.list, resolved),
        };
    });
}

export function useDeviceImages(deviceId: string | undefined) {
    return createQuery(() => ({
        queryKey: ["deviceImages", deviceId ?? ""],
        queryFn: () =>
            deviceId
                ? api.request(apiRoutes.images.listDevice, { device_id: deviceId })
                : Promise.resolve({ items: [], total: 0 }),
        enabled: !!deviceId,
    }));
}

export function useSetImageFavorite() {
    return createMutation(() => ({
        mutationFn: async (body: SetImageFavoriteRequest) => {
            await api.request(apiRoutes.images.favorite, { id: body.id, is_favorite: body.favorite });
        },
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["images"] }),
    }));
}

export function useBlacklistImage() {
    return createMutation(() => ({
        mutationFn: async (body: BlacklistImageRequest) => {
            await api.request(apiRoutes.images.blacklist, { image_id: body.id, reason: body.reason });
        },
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["images"] }),
    }));
}

export function useDeleteImage() {
    return createMutation(() => ({
        mutationFn: async (body: DeleteImageRequest) => {
            await api.request(apiRoutes.images.delete, body);
        },
        onSuccess: () => QueryClient.invalidateQueries({ queryKey: ["images"] }),
    }));
}

// ==================== Jobs ====================

export function useJobs(input: ListJobsRequest | (() => ListJobsRequest) = {}) {
    return createQuery(() => {
        const resolved = resolveInput(input);
        return {
            queryKey: ["jobs", resolved],
            queryFn: () => api.request(apiRoutes.jobs.list, resolved),
        };
    });
}

export function useJob(id: string | undefined) {
    return createQuery(() => ({
        queryKey: ["job", id ?? ""],
        queryFn: () => (id ? api.request(apiRoutes.jobs.get, { id }) : Promise.resolve(null)),
        enabled: !!id,
    }));
}

// ==================== Runtime Status ====================

export function useRuntimeStatus() {
    return createQuery(() => ({
        queryKey: ["runtimeStatus"],
        queryFn: () => api.request(apiRoutes.runtimeStatus.get, undefined as never),
        refetchInterval: 5000,
    }));
}
