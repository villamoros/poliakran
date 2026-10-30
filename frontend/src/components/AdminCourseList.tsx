"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { STATUS_LABELS, categoryLabel, type AdminCourse } from "@/lib/catalog";
import styles from "./ui.module.css";

const dateFmt = new Intl.DateTimeFormat("es-CO", { dateStyle: "medium" });

export function AdminCourseList() {
  const router = useRouter();
  const [courses, setCourses] = useState<AdminCourse[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    api
      .listCourses()
      .then((data) => active && setCourses(data.items))
      .catch((err) => {
        if (!active) return;
        if (err instanceof ApiError && err.status === 401) router.replace("/ingresar?next=/admin/cursos");
        else if (err instanceof ApiError && err.status === 403)
          setError("Tu cuenta no tiene permiso para gestionar cursos.");
        else setError("No pudimos cargar los cursos.");
      });
    return () => {
      active = false;
    };
  }, [router]);

  if (error)
    return (
      <div role="alert" className={styles.alert}>
        {error}
      </div>
    );
  if (!courses) return <p className={styles.subtitle}>Cargando cursos…</p>;
  if (courses.length === 0)
    return <p className={styles.empty}>Aún no hay cursos. Crea el primero para empezar el catálogo.</p>;

  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th scope="col">Curso</th>
          <th scope="col">Categoría</th>
          <th scope="col">Estado</th>
          <th scope="col">Actualizado</th>
        </tr>
      </thead>
      <tbody>
        {courses.map((c) => (
          <tr key={c.id}>
            <td>
              <Link href={`/admin/cursos/${c.id}`}>{c.title}</Link>
            </td>
            <td>{categoryLabel(c.category)}</td>
            <td>
              <span className={`${styles.status} ${styles[c.status]}`}>{STATUS_LABELS[c.status]}</span>
            </td>
            <td>{dateFmt.format(new Date(c.updatedAt))}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
