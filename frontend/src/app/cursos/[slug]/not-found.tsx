import Link from "next/link";
import styles from "@/components/ui.module.css";

export default function CourseNotFound() {
  return (
    <div className={styles.page}>
      <section className={styles.card}>
        <h1 className={styles.title}>Curso no disponible</h1>
        <p className={styles.subtitle}>Este curso no existe o ya no está publicado.</p>
        <Link href="/cursos" className={styles.button}>
          Ver el catálogo
        </Link>
      </section>
    </div>
  );
}
