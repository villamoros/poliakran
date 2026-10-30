import Link from "next/link";
import styles from "@/components/ui.module.css";

export default function HomePage() {
  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <h1 className={styles.title}>Aprende competencias digitales a tu ritmo</h1>
        <p className={styles.subtitle}>
          Cursos cortos organizados en módulos y lecciones, con evaluaciones y seguimiento de tu progreso.
        </p>
        <div className={styles.actions} style={{ justifyContent: "center" }}>
          <Link href="/cursos" className={styles.button}>
            Ver cursos
          </Link>
          <Link href="/registro" className={`${styles.button} ${styles.secondary}`}>
            Crear cuenta
          </Link>
          <Link href="/ingresar" className={`${styles.button} ${styles.secondary}`}>
            Iniciar sesión
          </Link>
        </div>
      </section>
    </div>
  );
}
