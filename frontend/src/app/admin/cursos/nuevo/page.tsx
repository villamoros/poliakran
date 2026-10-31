import type { Metadata } from "next";
import Link from "next/link";
import { NewCourse } from "@/components/NewCourse";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Nuevo curso" };

export default function NewCoursePage() {
  return (
    <div className={styles.section}>
      <Link href="/admin/cursos" className={styles.back}>
        ← Volver a la gestión de cursos
      </Link>
      <section className={`${styles.card} ${styles.wide}`}>
        <h1 className={styles.title}>Nuevo curso</h1>
        <p className={styles.subtitle}>Se guardará como borrador. Podrás editarlo antes de publicarlo.</p>
        <NewCourse />
      </section>
    </div>
  );
}
