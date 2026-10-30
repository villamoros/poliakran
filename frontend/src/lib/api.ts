// Cliente de la API del backend. Las peticiones van a /api/* en el mismo
// origen; la cookie de sesión HttpOnly viaja sola y nunca se lee desde JS.

export interface User {
  id: string;
  email: string;
  displayName: string;
  createdAt: string;
}

export interface Me {
  user: User;
  tenant: { id: string; name: string; kind: "personal" | "organization" };
  roles: string[];
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly fields: Record<string, string> = {},
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const GENERIC_ERROR = "No pudimos completar la operación. Inténtalo de nuevo.";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      ...init,
      credentials: "same-origin",
      headers: init.body ? { "Content-Type": "application/json", ...init.headers } : init.headers,
    });
  } catch {
    throw new ApiError(0, "network_error", "No hay conexión con el servidor.");
  }

  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const err = data?.error;
    throw new ApiError(res.status, err?.code ?? "unknown_error", err?.message ?? GENERIC_ERROR, err?.fields ?? {});
  }
  return data as T;
}

export const api = {
  register: (body: { email: string; displayName: string; password: string }) =>
    request<{ user: User }>("/api/v1/auth/register", { method: "POST", body: JSON.stringify(body) }),
  login: (body: { email: string; password: string }) =>
    request<{ user: User }>("/api/v1/auth/login", { method: "POST", body: JSON.stringify(body) }),
  logout: () => request<void>("/api/v1/auth/logout", { method: "POST" }),
  me: () => request<Me>("/api/v1/auth/me"),
};
