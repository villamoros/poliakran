import type { Metadata } from "next";
import Link from "next/link";
import { RegisterForm } from "@/components/RegisterForm";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Crear cuenta" };

export default function RegisterPage() {
  return (
    <div className={styles.page}>
      <section className={styles.card}>
        <h1 className={styles.title}>Crea tu cuenta</h1>
        <p className={styles.subtitle}>Regístrate para inscribirte en los cursos.</p>
        <RegisterForm />
        <p className={styles.footer}>
          ¿Ya tienes cuenta? <Link href="/ingresar">Inicia sesión</Link>
        </p>
      </section>
    </div>
  );
}
