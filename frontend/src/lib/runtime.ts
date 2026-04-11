// Runtime configuration injected by backend via window.__WALENS__
export interface WalensRuntimeConfig {
  basePath: string;
  apiBase: string;
}

declare global {
  interface Window {
    __WALENS__?: WalensRuntimeConfig;
  }
}

// Get runtime config from window.__WALENS__, with safe defaults
export function getRuntimeConfig(): WalensRuntimeConfig {
  return window.__WALENS__ ?? { basePath: '/', apiBase: '/api' };
}

// Derive the full API base URL
export function getApiUrl(): string {
  const { basePath, apiBase } = getRuntimeConfig();
  // apiBase is already prefixed with basePath in the backend
  return apiBase;
}

// Build a full URL given an API path
export function buildApiUrl(path: string): string {
  return `${getApiUrl()}${path}`;
}
