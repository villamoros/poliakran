import { describe, expect, it } from "vitest";
import { safeNextPath, validateLogin, validateRegister } from "./validation";

const valid = { email: "ana@example.com", displayName: "Ana Pérez", password: "contraseña-segura" };

describe("validateRegister", () => {
  it("acepta datos válidos", () => {
    expect(validateRegister(valid)).toEqual({});
  });

  it.each([
    ["email", { ...valid, email: "" }],
    ["email", { ...valid, email: "ana@" }],
    ["displayName", { ...valid, displayName: "  " }],
    ["displayName", { ...valid, displayName: "A" }],
    ["password", { ...valid, password: "corta" }],
    ["password", { ...valid, password: " ".repeat(12) }],
    ["password", { ...valid, password: "ANA@example.com" }],
    ["password", { ...valid, password: "x".repeat(129) }],
  ])("marca error en %s", (field, values) => {
    expect(validateRegister(values)).toHaveProperty(field);
  });
});

describe("validateLogin", () => {
  it("exige correo y contraseña", () => {
    expect(validateLogin({ email: "", password: "" })).toEqual({
      email: expect.any(String),
      password: expect.any(String),
    });
  });
});

describe("safeNextPath", () => {
  it.each([
    [null, "/panel"],
    ["/panel", "/panel"],
    ["/cursos/1?x=y", "/cursos/1?x=y"],
    ["https://evil.example", "/panel"],
    ["//evil.example", "/panel"],
    ["/\\evil.example", "/panel"],
    ["javascript:alert(1)", "/panel"],
  ])("%s → %s", (input, expected) => {
    expect(safeNextPath(input)).toBe(expected);
  });
});
