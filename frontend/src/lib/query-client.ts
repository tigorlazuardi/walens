/**
 * TanStack Svelte Query v6 singleton and provider.
 * Provides QueryClient singleton compatible with @tanstack/svelte-query v6 API.
 */

import { QueryClient as TQQueryClient } from '@tanstack/svelte-query';

export const QueryClient = new TQQueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
});

export { QueryClient as queryClient };
