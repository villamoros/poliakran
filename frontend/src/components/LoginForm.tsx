"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { safeNextPath, validateLogin, type FieldErrors } from "@/lib/validation";
import { TextField } from "./TextField";
import styles from "./ui.module.css";

export function LoginForm() {
  const router = useRouter();
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    const values = { email: String(data.get("email") ?? ""), password: String(data.get("password") ?? "") };

    const clientErrors = validateLogin(values);
    setErrors(clientErrors);
    setFormError("");
    if (Object.keys(clientErrors).length > 0) return;

    setSubmitting(true);
    try {
      await api.login(values);
      const next = new URLSearchParams(window.location.search).get("next");
      router.replace(safeNextPath(next));
      router.refresh();
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "No pudimos iniciar sesión. Inténtalo de nuevo.");
      setSubmitting(false);
    }
  }

  return (
    <form className={styles.form} onSubmit={onSubmit} noValidate>
      {formError && (
        <div role="alert" className={styles.alert}>
          {formError}
        </div>
      )}
      <TextField name="email" label="Correo electrónico" type="email" autoComplete="email" error={errors.email} />
      <TextField name="password" label="Contraseña" type="password" autoComplete="current-password" error={errors.password} />
      <button type="submit" className={styles.button} disabled={submitting}>
        {submitting ? "Ingresando…" : "Iniciar sesión"}
      </button>
    </form>
  );
}
