import type { Metadata } from "next";
import Link from "next/link";
import { AdminCourseList } from "@/components/AdminCourseList";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Gestión de cursos" };

// La autorización real la decide el backend; proxy.ts solo evita mostrar la
// página a quien claramente no tiene sesión.
export default function AdminCoursesPage() {
  return (
    <div className={styles.section}>
      <div className={styles.toolbar}>
        <h1 className={styles.title}>Gestión de cursos</h1>
        <Link href="/admin/cursos/nuevo" className={styles.button}>
          Nuevo curso
        </Link>
      </div>
      <section className={styles.card} style={{ maxWidth: "none" }}>
        <AdminCourseList />
      </section>
    </div>
  );
}
