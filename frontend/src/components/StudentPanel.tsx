"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError, type Me } from "@/lib/api";
import styles from "./ui.module.css";

const ROLE_LABELS: Record<string, string> = {
  student: "Estudiante",
  instructor: "Instructor",
  admin: "Administrador",
  owner: "Propietario",
};

export function StudentPanel() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [error, setError] = useState("");
  const [leaving, setLeaving] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .me()
      .then((data) => active && setMe(data))
      .catch((err) => {
        if (!active) return;
        if (err instanceof ApiError && err.status === 401) {
          router.replace("/ingresar?next=/panel");
        } else {
          setError("No pudimos cargar tu información.");
        }
      });
    return () => {
      active = false;
    };
  }, [router]);

  async function logout() {
    setLeaving(true);
    try {
      await api.logout();
    } finally {
      router.replace("/ingresar");
      router.refresh();
    }
  }

  if (error) return <div role="alert" className={styles.alert}>{error}</div>;
  if (!me) return <p className={styles.subtitle}>Cargando…</p>;

  return (
    <>
      <h1 className={styles.title}>Hola, {me.user.displayName}</h1>
      <p className={styles.subtitle}>Tu sesión está activa. Pronto verás aquí tus cursos y tu progreso.</p>
      <dl className={styles.dl}>
        <dt>Correo</dt>
        <dd>{me.user.email}</dd>
        <dt>Rol</dt>
        <dd>{me.roles.map((r) => ROLE_LABELS[r] ?? r).join(", ")}</dd>
        <dt>Espacio</dt>
        <dd>{me.tenant.kind === "personal" ? "Cuenta personal" : me.tenant.name}</dd>
      </dl>
      <button type="button" className={`${styles.button} ${styles.secondary}`} onClick={logout} disabled={leaving}>
        {leaving ? "Cerrando sesión…" : "Cerrar sesión"}
      </button>
    </>
  );
}
