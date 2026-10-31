import Link from "next/link";
import { Suspense } from "react";
import { notFound } from "next/navigation";
import { connection } from "next/server";
import { categoryLabel, durationLabel, levelLabel } from "@/lib/catalog";
import { fetchPublishedCourse } from "@/lib/catalog-server";
import styles from "@/components/ui.module.css";

// LMS-US-003: detalle de un curso publicado. Un borrador o un curso retirado
// responde 404 desde el backend y aquí se muestra la página de no encontrado.
export default function CourseDetailPage({ params }: PageProps<"/cursos/[slug]">) {
  return (
    <div className={styles.section}>
      <Link href="/cursos" className={styles.back}>
        ← Volver al catálogo
      </Link>
      <Suspense fallback={<p className={styles.subtitle}>Cargando curso…</p>}>
        {params.then(({ slug }) => (
          <CourseDetail slug={slug} />
        ))}
      </Suspense>
    </div>
  );
}

async function CourseDetail({ slug }: { slug: string }) {
  await connection();
  const course = await fetchPublishedCourse(slug);
  if (!course) notFound();

  return (
    <article className={`${styles.card} ${styles.wide} ${styles.detail}`}>
      <span className={styles.badge}>{categoryLabel(course.category)}</span>
      <h1 className={styles.title}>{course.title}</h1>
      <p className={styles.subtitle}>{course.summary}</p>
      <dl className={styles.dl}>
        <dt>Nivel</dt>
        <dd>{levelLabel(course.level)}</dd>
        <dt>Duración estimada</dt>
        <dd>{durationLabel(course.durationHours)}</dd>
      </dl>

      <h2 className={styles.h2}>Descripción</h2>
      <p className={styles.prose}>{course.description}</p>

      {course.objectives && course.objectives.length > 0 && (
        <>
          <h2 className={styles.h2}>Qué aprenderás</h2>
          <ul className={styles.list}>
            {course.objectives.map((o) => (
              <li key={o}>{o}</li>
            ))}
          </ul>
        </>
      )}

      {course.prerequisites && (
        <>
          <h2 className={styles.h2}>Requisitos previos</h2>
          <p className={styles.prose}>{course.prerequisites}</p>
        </>
      )}

      <p className={styles.footer}>
        La inscripción estará disponible próximamente. <Link href="/registro">Crea tu cuenta</Link> para no perdértela.
      </p>
    </article>
  );
}
