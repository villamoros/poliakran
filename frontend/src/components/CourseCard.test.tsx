import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { CourseCard } from "./CourseCard";

afterEach(cleanup);

describe("CourseCard", () => {
  it("presenta la información principal y enlaza al detalle", () => {
    render(
      <CourseCard
        course={{
          slug: "introduccion-a-go",
          title: "Introducción a Go",
          summary: "Fundamentos del lenguaje Go paso a paso.",
          category: "programacion",
          level: "basico",
          durationHours: 6,
          publishedAt: "2026-10-30T00:00:00Z",
        }}
      />,
    );
    expect(screen.getByRole("link", { name: "Introducción a Go" }).getAttribute("href")).toBe(
      "/cursos/introduccion-a-go",
    );
    expect(screen.getByText("Programación")).toBeTruthy();
    expect(screen.getByText("Básico · 6 horas")).toBeTruthy();
  });
});
