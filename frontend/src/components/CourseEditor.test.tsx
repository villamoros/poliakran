import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CourseEditor } from "./CourseEditor";
import { api, ApiError } from "@/lib/api";
import type { AdminCourse } from "@/lib/catalog";

const replace = vi.fn();
// Como en Next.js, el router es estable entre renders.
const router = { replace, refresh: vi.fn() };
vi.mock("next/navigation", () => ({ useRouter: () => router }));

const draft: AdminCourse = {
  id: "c1",
  slug: "curso-de-sql",
  title: "Curso de SQL",
  summary: "Consultas básicas para analizar datos.",
  description: "",
  category: "",
  level: "",
  durationHours: 0,
  objectives: [],
  prerequisites: "",
  status: "draft",
  publishedAt: null,
  withdrawnAt: null,
  createdAt: "2026-10-30T00:00:00Z",
  updatedAt: "2026-10-30T00:00:00Z",
  version: 1,
};

beforeEach(() => replace.mockReset());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("CourseEditor", () => {
  it("muestra qué falta cuando el curso no cumple las condiciones de publicación", async () => {
    vi.spyOn(api, "getCourse").mockResolvedValue({ course: draft });
    vi.spyOn(api, "publishCourse").mockRejectedValue(
      new ApiError(422, "publish_requirements_unmet", "El curso aún no cumple las condiciones para publicarse.", {
        description: "Para publicar, la descripción completa debe tener al menos 50 caracteres.",
      }),
    );
    render(<CourseEditor id="c1" />);
    await userEvent.setup().click(await screen.findByRole("button", { name: "Publicar" }));

    expect((await screen.findByRole("alert")).textContent).toContain("no cumple las condiciones");
    expect(screen.getByText(/al menos 50 caracteres/)).toBeTruthy();
    expect(screen.getByLabelText("Descripción completa").getAttribute("aria-invalid")).toBe("true");
  });

  it("publica y ofrece retirar el curso", async () => {
    vi.spyOn(api, "getCourse").mockResolvedValue({ course: draft });
    vi.spyOn(api, "publishCourse").mockResolvedValue({
      course: {
        ...draft,
        status: "published",
        publishedAt: "2026-10-30T01:00:00Z",
        version: 2,
      },
    });
    render(<CourseEditor id="c1" />);
    await userEvent.setup().click(await screen.findByRole("button", { name: "Publicar" }));

    expect((await screen.findByRole("status")).textContent).toContain("ya aparece en el catálogo");
    expect(screen.getByRole("button", { name: "Retirar del catálogo" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Ver en el catálogo" }).getAttribute("href")).toBe("/cursos/curso-de-sql");
  });

  it("guarda con la versión leída y avisa si otro cambio llegó antes", async () => {
    vi.spyOn(api, "getCourse").mockResolvedValue({ course: draft });
    const update = vi
      .spyOn(api, "updateCourse")
      .mockRejectedValue(
        new ApiError(
          409,
          "version_conflict",
          "El curso cambió mientras lo editabas. Recarga para ver la última versión.",
        ),
      );
    render(<CourseEditor id="c1" />);
    await userEvent.setup().click(await screen.findByRole("button", { name: "Guardar cambios" }));

    expect(update).toHaveBeenCalledWith("c1", 1, expect.objectContaining({ title: "Curso de SQL" }));
    expect((await screen.findByRole("alert")).textContent).toContain("cambió mientras lo editabas");
  });

  it("redirige a iniciar sesión si no hay sesión", async () => {
    vi.spyOn(api, "getCourse").mockRejectedValue(new ApiError(401, "unauthenticated", "Inicia sesión para continuar."));
    render(<CourseEditor id="c1" />);
    await vi.waitFor(() => expect(replace).toHaveBeenCalledWith("/ingresar?next=/admin/cursos/c1"));
  });
});
