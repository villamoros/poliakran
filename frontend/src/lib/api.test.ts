import { afterEach, describe, expect, it, vi } from "vitest";
import { api, ApiError, canManageCatalog } from "./api";

function mockFetch(status: number, body?: unknown) {
  const fn = vi.fn().mockResolvedValue(
    new Response(body === undefined ? null : JSON.stringify(body), { status }),
  );
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("api", () => {
  it("envía JSON al mismo origen con credenciales", async () => {
    const fetchMock = mockFetch(200, { user: { id: "1" } });
    await api.login({ email: "a@b.co", password: "x" });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/auth/login");
    expect(init.method).toBe("POST");
    expect(init.credentials).toBe("same-origin");
    expect(init.headers["Content-Type"]).toBe("application/json");
  });

  it("convierte el error del backend en ApiError con campos", async () => {
    mockFetch(422, {
      error: { code: "validation_failed", message: "Revisa los datos", fields: { email: "Inválido" } },
    });
    const err = await api.register({ email: "x", displayName: "y", password: "z" }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 422, code: "validation_failed", fields: { email: "Inválido" } });
  });

  it("acepta 204 sin cuerpo", async () => {
    mockFetch(204);
    await expect(api.logout()).resolves.toBeUndefined();
  });

  it("informa de errores de red", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(api.me()).rejects.toMatchObject({ code: "network_error" });
  });
});

describe("api de administración del catálogo", () => {
  it("envía la versión leída al editar", async () => {
    const fetchMock = mockFetch(200, { course: { id: "c1" } });
    const input = {
      title: "Curso",
      summary: "Resumen del curso de prueba",
      description: "",
      category: "",
      level: "",
      durationHours: 0,
      objectives: [],
      prerequisites: "",
    };
    await api.updateCourse("c1", 3, input);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/admin/courses/c1");
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body)).toEqual({ ...input, version: 3 });
  });

  it("publica y retira con POST sin cuerpo", async () => {
    const fetchMock = mockFetch(200, { course: { id: "c1" } });
    await api.publishCourse("c1");
    await api.withdrawCourse("c1");
    expect(fetchMock.mock.calls.map(([url, init]) => `${init.method} ${url}`)).toEqual([
      "POST /api/v1/admin/courses/c1/publish",
      "POST /api/v1/admin/courses/c1/withdraw",
    ]);
  });

  it("solo admin y owner gestionan el catálogo en la interfaz", () => {
    const me = (roles: string[]) => ({
      user: { id: "1", email: "", displayName: "", createdAt: "" },
      tenant: { id: "t", name: "", kind: "platform" as const },
      roles,
    });
    expect(canManageCatalog(me(["admin"]))).toBe(true);
    expect(canManageCatalog(me(["owner"]))).toBe(true);
    expect(canManageCatalog(me(["student"]))).toBe(false);
    expect(canManageCatalog(null)).toBe(false);
  });
});
