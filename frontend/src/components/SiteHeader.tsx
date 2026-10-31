import Link from "next/link";
import styles from "./ui.module.css";

export function SiteHeader() {
  return (
    <header className={styles.header}>
      <Link href="/" className={styles.brand}>
        Politécnico <span>Akran</span>
      </Link>
      <nav aria-label="Principal" className={styles.nav}>
        <Link href="/cursos">Cursos</Link>
        {/* Sin prefetch: sin sesión, proxy.ts redirige /panel a /ingresar y esa
            respuesta precargada se reutilizaría justo después de iniciar sesión. */}
        <Link href="/panel" prefetch={false}>
          Mi panel
        </Link>
      </nav>
    </header>
  );
}
