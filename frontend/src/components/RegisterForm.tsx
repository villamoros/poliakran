"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { MIN_PASSWORD_LENGTH, validateRegister, type FieldErrors } from "@/lib/validation";
import { TextField } from "./TextField";
import styles from "./ui.module.css";

export function RegisterForm() {
  const router = useRouter();
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    const values = {
      email: String(data.get("email") ?? ""),
      displayName: String(data.get("displayName") ?? ""),
      password: String(data.get("password") ?? ""),
    };

    const clientErrors = validateRegister(values);
    setErrors(clientErrors);
    setFormError("");
    if (Object.keys(clientErrors).length > 0) return;

    setSubmitting(true);
    try {
      await api.register(values);
      router.replace("/panel");
      router.refresh();
    } catch (err) {
      if (err instanceof ApiError) {
        setErrors(err.fields);
        setFormError(err.message);
      } else {
        setFormError("No pudimos crear tu cuenta. Inténtalo de nuevo.");
      }
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
      <TextField name="displayName" label="Nombre" autoComplete="name" error={errors.displayName} />
      <TextField name="email" label="Correo electrónico" type="email" autoComplete="email" error={errors.email} />
      <TextField
        name="password"
        label="Contraseña"
        type="password"
        autoComplete="new-password"
        hint={`Mínimo ${MIN_PASSWORD_LENGTH} caracteres.`}
        error={errors.password}
      />
      <button type="submit" className={styles.button} disabled={submitting}>
        {submitting ? "Creando cuenta…" : "Crear cuenta"}
      </button>
    </form>
  );
}
