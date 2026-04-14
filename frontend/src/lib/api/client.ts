/**
 * API client using openapi-fetch.
 *
 * All RPC endpoints use POST, with JSON body when required (Huma convention).
 * Auth handling: on 401 response, redirects browser to /login with
 * the current path preserved as a redirect target. After successful
 * login, the user is sent back to the original path.
 */

import createClient, { type Client } from "openapi-fetch";
import { buildAppPath, getRuntimeConfig } from "../runtime";
import { stripBasePath } from "../path";
import { apiRoutes } from "./routes";
import type { paths, ErrorModel } from "./types";

/** Union of all paths that support POST — used to constrain the request helper. */
type PostPaths = {
    [K in keyof paths]: paths[K] extends { post: unknown } ? K : never;
}[keyof paths];

/**
 * Extracts the JSON request body type for a given POST path.
 * Derives from `paths[P]['post']['requestBody']['content']['application/json']`.
 */
type PostRequestBody<P extends PostPaths> = paths[P] extends {
    post: { requestBody: { content: { "application/json": infer R } } };
}
    ? R
    : never;

type PostSuccessResponse<P extends PostPaths> = paths[P] extends {
    post: { responses: { 200: { content: { "application/json": infer R } } } };
}
    ? R
    : void;

type PostRequestArgs<P extends PostPaths> = PostRequestBody<P> extends never
    ? []
    : [body: PostRequestBody<P>];

type PostCaller = <P extends PostPaths>(
    path: P,
    init?: { body?: PostRequestBody<P> },
) => Promise<{ response: Response; data?: unknown }>;

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
        fetch: (input) => fetch(input, { credentials: "include" }),
    });
}

/** Default API client instance. */
export const apiClient = createApiClient(basePath);

const post = apiClient.POST as unknown as PostCaller;

// ==================== Auth Helpers ====================

/** Store the path to redirect to after login (session-scoped). */
let redirectAfterLogin = "/";

/**
 * Set the path to redirect to after a successful login.
 * Call this before navigating to /login.
 */
export function setRedirectAfterLogin(path: string) {
    redirectAfterLogin = path || "/";
}

/**
 * Get the stored redirect path and reset it.
 */
export function popRedirectAfterLogin(): string {
    const r = redirectAfterLogin;
    redirectAfterLogin = "/";
    return r;
}

// ==================== Internal 401 Handler ====================

/** Handles 401 by redirecting browser to login page. */
function handleAuthError() {
    const target = stripBasePath(window.location.pathname, basePath) || "/";
    const loginUrl = `${buildAppPath("login")}${target !== "/" ? "?redirect=" + encodeURIComponent(target) : ""}`;
    window.location.href = loginUrl;
    throw new Error("Unauthorized – redirecting to login");
}

// ==================== Internal Request Helper ====================

/**
 * Thin wrapper around `apiClient.POST` that handles 401 redirect and
 * common void/success patterns. Preserves schedule response unwrapping
 * behaviour in the individual functions that need it.
 */
export async function request<P extends PostPaths>(
    path: P,
    ...args: PostRequestArgs<P>
): Promise<PostSuccessResponse<P>> {
    const body = args[0];
    const res = body === undefined ? await post(path) : await post(path, { body });
    if (res.response.status === 401) handleAuthError();

    // Surface server-level error details when openapi-fetch exposes them.
    if (!res.response.ok) {
        // Cast to ErrorModel to access .detail since error responses are
        // always the "default" (error) schema regardless of success type T.
        const err = res.data as ErrorModel | undefined;
        const detail =
            err?.detail ??
            (err ? JSON.stringify(err) : res.response.statusText);
        throw new Error(detail || `Request failed: ${res.response.status}`);
    }

    // 204 No Content — nothing to return.
    if (res.response.status === 204) return undefined as PostSuccessResponse<P>;

    return res.data as PostSuccessResponse<P>;
}

// ==================== Auth Helpers ====================

export async function login(username: string, password: string): Promise<void> {
    await request(apiRoutes.login, { username, password });
}

export async function logout(): Promise<void> {
    await request(apiRoutes.logout);
}
