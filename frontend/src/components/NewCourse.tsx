"use client";

import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import type { CourseInput } from "@/lib/catalog";
import { CourseForm } from "./CourseForm";

export function NewCourse() {
  const router = useRouter();

  async function create(input: CourseInput) {
    try {
      const { course } = await api.createCourse(input);
      router.replace(`/admin/cursos/${course.id}`);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/ingresar?next=/admin/cursos/nuevo");
        return;
      }
      throw err;
    }
  }

  return <CourseForm submitLabel="Crear borrador" submittingLabel="Creando…" onSubmit={create} />;
}
