import type { Metadata } from "next";
import { StudentPanel } from "@/components/StudentPanel";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Mi panel" };

// La autorización real la decide el backend en /api/v1/auth/me; proxy.ts solo
// evita mostrar la página a quien claramente no tiene sesión.
export default function PanelPage() {
  return (
    <div className={styles.page}>
      <section className={`${styles.card} ${styles.wide}`}>
        <StudentPanel />
      </section>
    </div>
  );
}
