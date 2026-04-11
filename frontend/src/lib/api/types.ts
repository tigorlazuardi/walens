/**
 * Frontend API client types aligned with actual backend Huma routes.
 * These types represent the JSON shapes that travel over the wire.
 *
 * Key backend type mappings:
 * - UUID: stored as {"uuid": "hex-string"}, but sent as plain string in requests
 * - BoolInt: serialized as boolean in JSON
 * - UnixMilliTime: serialized as RFC3339 string in JSON
 * - RawJSON: serialized as raw JSON object in JSON
 */

// ==================== Common Types ====================

export interface CursorPaginationRequest {
  limit?: number;
  cursor?: string;
  direction?: 'next' | 'prev';
}

export interface CursorPaginationResponse {
  next?: string;
  prev?: string;
}

// ==================== Config (Persisted) ====================
// Persisted config - only data_dir and log_level

export interface PersistedConfig {
  data_dir: string;
  log_level: string;
}

export interface UpdateConfigRequest {
  data_dir: string;
  log_level: string;
}

// ==================== Devices ====================

export interface Device {
  id: string; // UUID string
  name: string;
  slug: string;
  screen_width: number;
  screen_height: number;
  min_image_width: number;
  max_image_width: number;
  min_image_height: number;
  max_image_height: number;
  min_filesize: number;
  max_filesize: number;
  is_adult_allowed: boolean;
  is_enabled: boolean;
  aspect_ratio_tolerance: number;
  created_at: string; // RFC3339
  updated_at: string; // RFC3339
}

export interface ListDevicesRequest {
  search?: string;
  pagination?: CursorPaginationRequest;
}

export interface ListDevicesResponse {
  items: Device[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface CreateDeviceRequest {
  name: string;
  slug: string;
  screen_width: number;
  screen_height: number;
  min_image_width?: number;
  max_image_width?: number;
  min_image_height?: number;
  max_image_height?: number;
  min_filesize?: number;
  max_filesize?: number;
  is_adult_allowed?: boolean;
  is_enabled?: boolean;
  aspect_ratio_tolerance?: number;
}

export interface UpdateDeviceRequest extends CreateDeviceRequest {
  id: string;
}

// ==================== Sources ====================

export interface Source {
  id: string; // UUID string
  name: string;
  source_type: string;
  params: Record<string, unknown>;
  lookup_count: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListSourcesRequest {
  search?: string;
  pagination?: CursorPaginationRequest;
}

export interface ListSourcesResponse {
  items: Source[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface CreateSourceRequest {
  name: string;
  source_type: string;
  params?: Record<string, unknown>;
  lookup_count?: number;
  is_enabled?: boolean;
}

export interface UpdateSourceRequest {
  id: string;
  name?: string;
  source_type?: string;
  params?: Record<string, unknown>;
  lookup_count?: number;
  is_enabled?: boolean;
}

// ==================== Source Types ====================

export interface SourceType {
  type_name: string;
  display_name: string;
}

export interface ListSourceTypesResponse {
  items: SourceType[];
}

// ==================== Source Schedules ====================

export interface SourceSchedule {
  id: string;
  source_id: string;
  source_name?: string; // joined field from ScheduleWithSource
  cron_expr: string;
  is_enabled: boolean;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ListSchedulesRequest {
  source_id?: string;
  pagination?: CursorPaginationRequest;
}

export interface ListSchedulesResponse {
  items: SourceSchedule[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface CreateScheduleRequest {
  source_id: string;
  cron_expr: string;
  is_enabled?: boolean;
}

export interface UpdateScheduleRequest {
  id: string;
  source_id?: string;
  cron_expr?: string;
  is_enabled?: boolean;
}

// ==================== Device Subscriptions ====================

export interface DeviceSubscription {
  id: string;
  device_id: string;
  source_id: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListSubscriptionsRequest {
  pagination?: CursorPaginationRequest;
}

export interface ListSubscriptionsResponse {
  items: DeviceSubscription[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface CreateSubscriptionRequest {
  device_id: string;
  source_id: string;
  is_enabled?: boolean;
}

export interface UpdateSubscriptionRequest {
  id: string;
  device_id?: string;
  source_id?: string;
  is_enabled?: boolean;
}

// ==================== Images ====================

export interface Image {
  id: string;
  source_id: string;
  unique_identifier: string;
  source_type: string;
  original_filename?: string;
  preview_url?: string;
  origin_url?: string;
  source_item_identifier?: string;
  original_identifier?: string;
  uploader?: string;
  artist?: string;
  mime_type?: string;
  file_size_bytes?: number;
  width?: number;
  height?: number;
  aspect_ratio?: number;
  is_adult: boolean;
  is_favorite: boolean;
  json_meta?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ListImagesRequest {
  adult?: boolean;
  favorite?: boolean;
  min_width?: number;
  max_width?: number;
  min_height?: number;
  max_height?: number;
  min_file_size_bytes?: number;
  max_file_size_bytes?: number;
  search?: string;
  source_ids?: string[];
  pagination?: CursorPaginationRequest;
}

export interface ListImagesResponse {
  items: Image[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface ListDeviceImagesRequest {
  device_id: string;
  pagination?: CursorPaginationRequest;
}

export interface SetImageFavoriteRequest {
  id: string;
  favorite: boolean;
}

export interface BlacklistImageRequest {
  id: string;
}

export interface DeleteImageRequest {
  id: string;
}

// ==================== Jobs ====================

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';
export type JobType = 'source_sync' | 'source_download';
export type TriggerKind = 'manual' | 'schedule' | 'recovery';

export interface Job {
  id: string;
  job_type: JobType;
  source_id?: string;
  source_name?: string;
  source_type?: string;
  status: JobStatus;
  trigger_kind: TriggerKind;
  run_after: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  requested_image_count: number;
  downloaded_image_count: number;
  reused_image_count: number;
  hardlinked_image_count: number;
  copied_image_count: number;
  stored_image_count: number;
  skipped_image_count: number;
  message?: string;
  error_message?: string;
  json_input?: Record<string, unknown>;
  json_result?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ListJobsRequest {
  status?: JobStatus;
  job_type?: JobType;
  source_id?: string;
  trigger_kind?: TriggerKind;
  pagination?: CursorPaginationRequest;
}

export interface ListJobsResponse {
  items: Job[];
  pagination?: CursorPaginationResponse;
  total: number;
}

export interface GetJobRequest {
  id: string;
}

// ==================== Runtime Status ====================

export interface RuntimeStatus {
  status: 'ok' | 'degraded' | 'stopping';
  queue_size: number;
  scheduler_ready: boolean;
  schedule_count: number;
  runner_active: boolean;
}
