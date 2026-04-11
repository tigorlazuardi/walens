/**
 * API type aliases derived from the OpenAPI-generated schema.
 * Re-exports friendly names for use throughout the frontend app.
 */

import type {
  paths,
  components,
} from './generated/schema';

// ==================== Common Types ====================

export type CursorPaginationRequest = components['schemas']['CursorPaginationRequest'];
export type CursorPaginationResponse = components['schemas']['CursorPaginationResponse'];

// ==================== Config (Persisted) ====================

export type PersistedConfig = components['schemas']['PersistedConfig'];
export type UpdateConfigRequest = components['schemas']['UpdateConfigRequest'];

// ==================== Devices ====================

export type Device = components['schemas']['Devices'];
export type ListDevicesRequest = components['schemas']['ListDevicesRequest'];
export type ListDevicesResponse = components['schemas']['ListDevicesResponse'];
export type CreateDeviceRequest = components['schemas']['CreateDeviceRequest'];
export type UpdateDeviceRequest = components['schemas']['UpdateDeviceRequest'];

// ==================== Sources ====================

export type Source = components['schemas']['Sources'];
export type ListSourcesRequest = components['schemas']['ListSourcesRequest'];
export type ListSourcesResponse = components['schemas']['ListSourcesResponse'];
export type CreateSourceRequest = components['schemas']['CreateSourceRequest'];
export type UpdateSourceRequest = components['schemas']['UpdateSourceRequest'];

// ==================== Source Types ====================

export type SourceType = components['schemas']['SourceTypeMetadata'];
export type ListSourceTypesResponse = components['schemas']['ListSourceTypesOutputBody'];

// ==================== Source Schedules ====================

export type SourceSchedule = components['schemas']['SourceSchedules'];
export type ListSchedulesRequest = components['schemas']['ListSchedulesRequest'];
export type ListSchedulesResponse = components['schemas']['ListSchedulesResponse'];
export type CreateScheduleRequest = components['schemas']['CreateScheduleRequest'];
export type UpdateScheduleRequest = components['schemas']['UpdateScheduleRequest'];

// ==================== Device Subscriptions ====================

export type DeviceSubscription = components['schemas']['DeviceSourceSubscriptions'];
export type ListSubscriptionsRequest = components['schemas']['ListSubscriptionsRequest'];
export type ListSubscriptionsResponse = components['schemas']['ListSubscriptionsResponse'];
export type CreateSubscriptionRequest = components['schemas']['CreateSubscriptionRequest'];
export type UpdateSubscriptionRequest = components['schemas']['UpdateSubscriptionRequest'];

// ==================== Images ====================

export type Image = components['schemas']['Images'];
export type ListImagesRequest = components['schemas']['ListImagesRequest'];
export type ListImagesResponse = components['schemas']['ListImagesResponse'];
export type ListDeviceImagesRequest = components['schemas']['ListDeviceImagesRequest'];
// ==================== Image Mutations (App-facing wrappers) ====================
// These use friendlier field names than the generated schema.
// The client maps them to the generated field names before sending.

// App-facing: uses `favorite` instead of `is_favorite`
export interface SetImageFavoriteRequest {
  id: string;
  favorite: boolean;
}

// App-facing: uses `id` instead of `image_id`
export interface BlacklistImageRequest {
  id: string;
  reason?: string | null;
}

// ==================== Image Mutations (Generated schema aliases) ====================
// Original generated schema shapes — used internally by the client.
export type SetImageFavoriteInput = components['schemas']['SetImageFavoriteInput'];
export type BlacklistImageInput = components['schemas']['BlacklistImageInput'];
export type DeleteImageRequest = components['schemas']['DeleteImageInput'];

// ==================== Jobs ====================

export type Job = components['schemas']['Jobs'];
export type ListJobsRequest = components['schemas']['ListJobsRequest'];
export type ListJobsResponse = components['schemas']['ListJobsResponse'];
export type GetJobRequest = components['schemas']['GetJobRequest'];

/** Job status: queued, running, succeeded, failed, cancelled */
export type JobStatus = string;
/** Job type: source_sync, source_download */
export type JobType = string;
/** Trigger kind: manual, schedule, recovery */
export type TriggerKind = string;

// ==================== Runtime Status ====================

export type RuntimeStatus = components['schemas']['RuntimeStatusOutputBody'];

// Expose paths type for openapi-fetch
export type { paths };

// ==================== Error Types ====================

/** Standard error response shape from the API. */
export type ErrorModel = components["schemas"]["ErrorModel"];
