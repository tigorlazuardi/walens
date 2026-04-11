export function normalizeBasePath(basePath = '/'): string {
  if (!basePath || basePath === '/') return '/';
  const withLeadingSlash = basePath.startsWith('/') ? basePath : `/${basePath}`;
  return withLeadingSlash.replace(/\/+$/, '') || '/';
}

export function stripBasePath(pathname: string, basePath = '/'): string {
  const normalizedBase = normalizeBasePath(basePath);
  const cleanPath = pathname || '/';

  if (normalizedBase === '/') {
    return cleanPath.startsWith('/') ? cleanPath : `/${cleanPath}`;
  }

  if (cleanPath === normalizedBase) return '/';
  if (cleanPath.startsWith(`${normalizedBase}/`)) {
    const stripped = cleanPath.slice(normalizedBase.length);
    return stripped.startsWith('/') ? stripped : `/${stripped}`;
  }

  return cleanPath.startsWith('/') ? cleanPath : `/${cleanPath}`;
}

export function joinBasePath(basePath: string, ...segments: string[]): string {
  const normalizedBase = normalizeBasePath(basePath);
  const suffix = segments
    .flatMap((segment) => segment.split('/'))
    .map((segment) => segment.trim())
    .filter(Boolean)
    .join('/');

  if (!suffix) return normalizedBase;
  if (normalizedBase === '/') return `/${suffix}`;
  return `${normalizedBase}/${suffix}`;
}

export function normalizeAppPath(pathname: string): string {
  if (!pathname) return '/';
  return pathname.startsWith('/') ? pathname : `/${pathname}`;
}
