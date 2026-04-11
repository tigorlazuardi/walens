export function rpcPath<Path extends string>(path: Path): `/api/${Path}` {
  return `/api/${path}` as `/api/${Path}`;
}

export const apiRoutes = {
  login: '/api/login',
  logout: '/api/logout',
  configs: {
    get: rpcPath('v1/configs/GetConfig'),
    update: rpcPath('v1/configs/UpdateConfig'),
  },
  devices: {
    list: rpcPath('v1/devices/ListDevices'),
    get: rpcPath('v1/devices/GetDevice'),
    create: rpcPath('v1/devices/CreateDevice'),
    update: rpcPath('v1/devices/UpdateDevice'),
    delete: rpcPath('v1/devices/DeleteDevice'),
  },
  sources: {
    list: rpcPath('v1/sources/ListSources'),
    get: rpcPath('v1/sources/GetSource'),
    create: rpcPath('v1/sources/CreateSource'),
    update: rpcPath('v1/sources/UpdateSource'),
    delete: rpcPath('v1/sources/DeleteSource'),
  },
  sourceTypes: {
    list: rpcPath('v1/source_types/ListSourceTypes'),
  },
  schedules: {
    list: rpcPath('v1/source_schedules/ListSourceSchedules'),
    get: rpcPath('v1/source_schedules/GetSourceSchedule'),
    create: rpcPath('v1/source_schedules/CreateSourceSchedule'),
    update: rpcPath('v1/source_schedules/UpdateSourceSchedule'),
    delete: rpcPath('v1/source_schedules/DeleteSourceSchedule'),
  },
  subscriptions: {
    list: rpcPath('v1/device_subscriptions/ListDeviceSubscriptions'),
    create: rpcPath('v1/device_subscriptions/CreateDeviceSubscription'),
    update: rpcPath('v1/device_subscriptions/UpdateDeviceSubscription'),
    delete: rpcPath('v1/device_subscriptions/DeleteDeviceSubscription'),
  },
  images: {
    list: rpcPath('v1/images/ListImages'),
    listDevice: rpcPath('v1/images/ListDeviceImages'),
    favorite: rpcPath('v1/images/SetImageFavorite'),
    blacklist: rpcPath('v1/images/BlacklistImage'),
    delete: rpcPath('v1/images/DeleteImage'),
  },
  jobs: {
    list: rpcPath('v1/jobs/ListJobs'),
    get: rpcPath('v1/jobs/GetJob'),
  },
  runtimeStatus: {
    get: rpcPath('v1/runtime_status/GetRuntimeStatus'),
  },
} as const;
