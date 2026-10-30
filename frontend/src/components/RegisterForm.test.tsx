import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RegisterForm } from "./RegisterForm";
import { api, ApiError } from "@/lib/api";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace, refresh: vi.fn() }) }));

beforeEach(() => replace.mockReset());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

async function fill(values: { name?: string; email?: string; password?: string }) {
  const user = userEvent.setup();
  if (values.name) await user.type(screen.getByLabelText("Nombre"), values.name);
  if (values.email) await user.type(screen.getByLabelText("Correo electrónico"), values.email);
  if (values.password) await user.type(screen.getByLabelText("Contraseña"), values.password);
  await user.click(screen.getByRole("button", { name: "Crear cuenta" }));
}

describe("RegisterForm", () => {
  it("muestra errores de validación sin llamar a la API", async () => {
    const spy = vi.spyOn(api, "register");
    render(<RegisterForm />);
    await fill({});
    expect(screen.getByText("Escribe tu nombre.")).toBeTruthy();
    expect(screen.getByLabelText("Correo electrónico").getAttribute("aria-invalid")).toBe("true");
    expect(spy).not.toHaveBeenCalled();
  });

  it("registra y redirige al panel", async () => {
    const spy = vi.spyOn(api, "register").mockResolvedValue({
      user: { id: "1", email: "ana@example.com", displayName: "Ana", createdAt: "" },
    });
    render(<RegisterForm />);
    await fill({ name: "Ana", email: "ana@example.com", password: "contraseña-segura" });
    expect(spy).toHaveBeenCalledWith({ displayName: "Ana", email: "ana@example.com", password: "contraseña-segura" });
    expect(replace).toHaveBeenCalledWith("/panel");
  });

  it("muestra el error del servidor cuando el correo ya existe", async () => {
    vi.spyOn(api, "register").mockRejectedValue(
      new ApiError(409, "email_taken", "Ya existe una cuenta con ese correo.", { email: "Este correo ya está registrado." }),
    );
    render(<RegisterForm />);
    await fill({ name: "Ana", email: "ana@example.com", password: "contraseña-segura" });
    expect((await screen.findByRole("alert")).textContent).toContain("Ya existe una cuenta");
    expect(screen.getByText("Este correo ya está registrado.")).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });
});
