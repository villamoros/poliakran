import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CourseForm } from "./CourseForm";
import { ApiError } from "@/lib/api";

afterEach(cleanup);

function renderForm(onSubmit = vi.fn().mockResolvedValue(undefined)) {
  render(<CourseForm submitLabel="Crear borrador" submittingLabel="Creando…" onSubmit={onSubmit} />);
  return onSubmit;
}

describe("CourseForm", () => {
  it("valida los campos obligatorios sin llamar a la API", async () => {
    const onSubmit = renderForm();
    await userEvent.setup().click(screen.getByRole("button", { name: "Crear borrador" }));
    expect(screen.getByText("Escribe el título del curso.")).toBeTruthy();
    expect(screen.getByLabelText("Descripción breve").getAttribute("aria-invalid")).toBe("true");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("envía los datos normalizados, con un objetivo por línea", async () => {
    const onSubmit = renderForm();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText("Título"), "Introducción a Go");
    await user.type(screen.getByLabelText("Descripción breve"), "Fundamentos del lenguaje Go paso a paso.");
    await user.selectOptions(screen.getByLabelText("Categoría"), "programacion");
    await user.type(screen.getByLabelText("Duración (horas)"), "6");
    await user.type(screen.getByLabelText("Objetivos de aprendizaje"), "Escribir funciones{enter}{enter}Usar módulos");
    await user.click(screen.getByRole("button", { name: "Crear borrador" }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Introducción a Go",
        category: "programacion",
        level: "",
        durationHours: 6,
        objectives: ["Escribir funciones", "Usar módulos"],
      }),
    );
  });

  it("muestra los errores por campo que devuelve el backend", async () => {
    const onSubmit = vi
      .fn()
      .mockRejectedValue(
        new ApiError(422, "validation_failed", "Revisa los datos del formulario.", {
          category: "Elige una categoría de la lista.",
        }),
      );
    renderForm(onSubmit);
    const user = userEvent.setup();
    await user.type(screen.getByLabelText("Título"), "Introducción a Go");
    await user.type(screen.getByLabelText("Descripción breve"), "Fundamentos del lenguaje Go paso a paso.");
    await user.click(screen.getByRole("button", { name: "Crear borrador" }));

    expect((await screen.findByRole("alert")).textContent).toContain("Revisa los datos");
    expect(screen.getByText("Elige una categoría de la lista.")).toBeTruthy();
  });
});
