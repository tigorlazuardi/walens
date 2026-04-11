/**
 * Simple reactive query/mutation hooks for Svelte 5 with runes.
 * Provides basic query/mutation functionality without TanStack Query dependency.
 */

export class QueryError extends Error {
  constructor(public originalError: unknown) {
    super(originalError instanceof Error ? originalError.message : 'Query failed');
    this.name = 'QueryError';
  }
}

// ==================== Query ====================

interface QueryResult<T> {
  data: T | undefined;
  isLoading: boolean;
  isError: boolean;
  error: QueryError | null;
}

export function createQuery<T>(options: {
  queryKey: unknown[];
  queryFn: () => Promise<T>;
  refetchInterval?: number;
}) {
  // Reactive state
  let data = $state<T | undefined>(undefined);
  let isLoading = $state(true);
  let isError = $state(false);
  let error = $state<QueryError | null>(null);

  let intervalId: ReturnType<typeof setInterval> | null = null;

  async function execute() {
    isLoading = true;
    isError = false;
    error = null;
    try {
      data = await options.queryFn();
    } catch (err) {
      isError = true;
      error = err instanceof QueryError ? err : new QueryError(err);
    }
    isLoading = false;
  }

  function refresh() {
    execute();
  }

  // Auto-execute on mount with optional refetch
  $effect(() => {
    execute();
    if (options.refetchInterval) {
      intervalId = setInterval(execute, options.refetchInterval);
    }
    return () => {
      if (intervalId) clearInterval(intervalId);
    };
  });

  return {
    get data() { return data; },
    get isLoading() { return isLoading; },
    get isError() { return isError; },
    get error() { return error; },
    refetch: refresh,
  };
}

// ==================== Mutation ====================

interface MutationState<TOutput> {
  isPending: boolean;
  isError: boolean;
  error: QueryError | null;
  data: TOutput | null;
}

export function createMutation<TInput, TOutput>(options: {
  mutationFn: (input: TInput) => Promise<TOutput>;
  onSuccess?: () => void;
}) {
  let state = $state<MutationState<TOutput>>({
    isPending: false,
    isError: false,
    error: null,
    data: null,
  });

  async function mutateAsync(input: TInput): Promise<TOutput> {
    state = { ...state, isPending: true, isError: false, error: null };
    try {
      const result = await options.mutationFn(input);
      state = { ...state, isPending: false, data: result };
      options.onSuccess?.();
      return result;
    } catch (err) {
      const qerr = err instanceof QueryError ? err : new QueryError(err);
      state = { ...state, isPending: false, isError: true, error: qerr };
      throw qerr;
    }
  }

  return {
    get isPending() { return state.isPending; },
    get isError() { return state.isError; },
    get error() { return state.error; },
    get data() { return state.data; },
    mutateAsync,
  };
}

// ==================== Query Invalidation ====================

// Simple global query cache with invalidation
const queryInvalidators = new Map<string, Set<() => void>>();

export function invalidateQueries(queryKey: unknown[]) {
  const key = JSON.stringify(queryKey);
  queryInvalidators.get(key)?.forEach(fn => fn());
}

export function registerQuery(key: unknown[], invalidate: () => void) {
  const strKey = JSON.stringify(key);
  if (!queryInvalidators.has(strKey)) {
    queryInvalidators.set(strKey, new Set());
  }
  queryInvalidators.get(strKey)!.add(invalidate);
}
