import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { CourseEditor } from "@/components/CourseEditor";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Editar curso" };

export default function EditCoursePage({ params }: PageProps<"/admin/cursos/[id]">) {
  return (
    <div className={styles.section}>
      <Link href="/admin/cursos" className={styles.back}>
        ← Volver a la gestión de cursos
      </Link>
      <section className={`${styles.card} ${styles.wide}`}>
        <Suspense fallback={<p className={styles.subtitle}>Cargando curso…</p>}>
          {params.then(({ id }) => (
            <CourseEditor id={id} />
          ))}
        </Suspense>
      </section>
    </div>
  );
}
