import { describe, expect, it } from "vitest";
import { safeNextPath, validateLogin, validateRegister, validateCourse } from "./validation";

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

describe("validateCourse", () => {
  const base = {
    title: "Introducción a Go",
    summary: "Fundamentos del lenguaje Go paso a paso.",
    description: "",
    durationHours: 0,
    objectives: [],
    prerequisites: "",
  };

  it("acepta un borrador con título y descripción breve", () => {
    expect(validateCourse(base)).toEqual({});
  });

  it("exige título y descripción breve", () => {
    const errors = validateCourse({ ...base, title: "  ", summary: "" });
    expect(errors.title).toBe("Escribe el título del curso.");
    expect(errors.summary).toBe("Escribe una descripción breve.");
  });

  it("valida longitudes, duración y objetivos", () => {
    const errors = validateCourse({
      ...base,
      title: "Go",
      summary: "corta",
      durationHours: 501,
      objectives: ["ok"],
    });
    expect(Object.keys(errors).sort()).toEqual(["durationHours", "objectives", "summary", "title"]);
  });

  it("limita a 10 objetivos", () => {
    const objectives = Array.from({ length: 11 }, (_, i) => `Objetivo ${i + 1}`);
    expect(validateCourse({ ...base, objectives }).objectives).toBe("Indica como máximo 10 objetivos.");
  });
});
