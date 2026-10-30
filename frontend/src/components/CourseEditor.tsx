"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { STATUS_LABELS, type AdminCourse } from "@/lib/catalog";
import type { FieldErrors } from "@/lib/validation";
import { CourseForm } from "./CourseForm";
import styles from "./ui.module.css";

// Edición, publicación y retiro de un curso (LMS-US-004 y LMS-US-006).
export function CourseEditor({ id }: { id: string }) {
  const router = useRouter();
  const [course, setCourse] = useState<AdminCourse | null>(null);
  const [loadError, setLoadError] = useState("");
  const [notice, setNotice] = useState("");
  const [actionError, setActionError] = useState("");
  const [publishErrors, setPublishErrors] = useState<FieldErrors>({});
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .getCourse(id)
      .then((data) => active && setCourse(data.course))
      .catch((err) => {
        if (!active) return;
        if (err instanceof ApiError && err.status === 401) router.replace(`/ingresar?next=/admin/cursos/${id}`);
        else if (err instanceof ApiError && err.status === 403)
          setLoadError("Tu cuenta no tiene permiso para gestionar cursos.");
        else if (err instanceof ApiError && err.status === 404) setLoadError("Este curso no existe.");
        else setLoadError("No pudimos cargar el curso.");
      });
    return () => {
      active = false;
    };
  }, [id, router]);

  if (loadError)
    return (
      <div role="alert" className={styles.alert}>
        {loadError}
      </div>
    );
  if (!course) return <p className={styles.subtitle}>Cargando curso…</p>;

  async function save(input: Parameters<typeof api.updateCourse>[2]) {
    setNotice("");
    setActionError("");
    setPublishErrors({});
    const { course: updated } = await api.updateCourse(course!.id, course!.version, input);
    setCourse(updated);
    setNotice("Cambios guardados.");
  }

  async function changeStatus(op: "publish" | "withdraw") {
    setBusy(true);
    setNotice("");
    setActionError("");
    setPublishErrors({});
    try {
      const { course: updated } =
        op === "publish" ? await api.publishCourse(course!.id) : await api.withdrawCourse(course!.id);
      setCourse(updated);
      setNotice(op === "publish" ? "Curso publicado: ya aparece en el catálogo." : "Curso retirado del catálogo.");
    } catch (err) {
      if (err instanceof ApiError) {
        setActionError(err.message);
        setPublishErrors(err.fields);
      } else {
        setActionError("No pudimos cambiar el estado del curso.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className={styles.toolbar}>
        <div>
          <h1 className={styles.title}>{course.title}</h1>
          <p className={styles.meta}>
            <span className={`${styles.status} ${styles[course.status]}`}>{STATUS_LABELS[course.status]}</span>
            {course.status === "published" && (
              <>
                {" "}
                · <Link href={`/cursos/${course.slug}`}>Ver en el catálogo</Link>
              </>
            )}
          </p>
        </div>
        <div className={styles.actions}>
          {course.status !== "published" ? (
            <button type="button" className={styles.button} onClick={() => changeStatus("publish")} disabled={busy}>
              {busy ? "Publicando…" : "Publicar"}
            </button>
          ) : (
            <button
              type="button"
              className={`${styles.button} ${styles.secondary}`}
              onClick={() => changeStatus("withdraw")}
              disabled={busy}
            >
              {busy ? "Retirando…" : "Retirar del catálogo"}
            </button>
          )}
        </div>
      </div>
      {notice && (
        <div role="status" className={styles.success}>
          {notice}
        </div>
      )}
      {actionError && (
        <div role="alert" className={styles.alert}>
          {actionError}
        </div>
      )}
      <CourseForm
        key={course.version}
        initial={course}
        submitLabel="Guardar cambios"
        submittingLabel="Guardando…"
        onSubmit={save}
        externalErrors={publishErrors}
      />
    </>
  );
}
