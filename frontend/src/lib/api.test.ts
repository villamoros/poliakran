import { afterEach, describe, expect, it, vi } from "vitest";
import { api, ApiError } from "./api";

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
