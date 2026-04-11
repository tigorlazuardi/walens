import { joinBasePath, normalizeBasePath } from './path';

export interface WalensRuntimeConfig {
  basePath: string;
  apiBase: string;
}

declare global {
  interface Window {
    __WALENS__?: WalensRuntimeConfig;
  }
}

export function getRuntimeConfig(): WalensRuntimeConfig {
  const runtime = window.__WALENS__;
  const basePath = normalizeBasePath(runtime?.basePath ?? '/');
  const apiBase = normalizeBasePath(runtime?.apiBase ?? joinBasePath(basePath, 'api'));

  return { basePath, apiBase };
}

export function buildAppPath(...segments: string[]): string {
  return joinBasePath(getRuntimeConfig().basePath, ...segments);
}

export function buildApiPath(...segments: string[]): string {
  return joinBasePath(getRuntimeConfig().apiBase, ...segments);
}
