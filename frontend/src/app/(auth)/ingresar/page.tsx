import type { Metadata } from "next";
import Link from "next/link";
import { LoginForm } from "@/components/LoginForm";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Iniciar sesión" };

export default function LoginPage() {
  return (
    <div className={styles.page}>
      <section className={styles.card}>
        <h1 className={styles.title}>Inicia sesión</h1>
        <p className={styles.subtitle}>Continúa con tu aprendizaje.</p>
        <LoginForm />
        <p className={styles.footer}>
          ¿No tienes cuenta? <Link href="/registro">Crea una</Link>
        </p>
      </section>
    </div>
  );
}
