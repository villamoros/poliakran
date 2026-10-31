"use client";

import { useState } from "react";
import { ApiError } from "@/lib/api";
import { CATEGORIES, LEVELS, type AdminCourse, type CourseInput } from "@/lib/catalog";
import { validateCourse, type FieldErrors } from "@/lib/validation";
import { SelectField } from "./SelectField";
import { TextArea } from "./TextArea";
import { TextField } from "./TextField";
import styles from "./ui.module.css";

interface Props {
  initial?: AdminCourse;
  submitLabel: string;
  submittingLabel: string;
  onSubmit: (input: CourseInput) => Promise<void>;
  /** Errores que llegan de fuera del formulario (p. ej. al publicar). */
  externalErrors?: FieldErrors;
}

function readForm(form: HTMLFormElement): CourseInput {
  const data = new FormData(form);
  const text = (k: string) => String(data.get(k) ?? "");
  const hours = text("durationHours").trim();
  return {
    title: text("title"),
    summary: text("summary"),
    description: text("description"),
    category: text("category"),
    level: text("level"),
    durationHours: hours === "" ? 0 : Number(hours),
    objectives: text("objectives")
      .split("\n")
      .map((o) => o.trim())
      .filter(Boolean),
    prerequisites: text("prerequisites"),
  };
}

// Formulario de creación y edición de un curso (LMS-US-004). La validación del
// cliente da respuesta inmediata; la autoridad sigue siendo el backend.
export function CourseForm({ initial, submitLabel, submittingLabel, onSubmit, externalErrors = {} }: Props) {
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const shown = { ...externalErrors, ...errors };

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const input = readForm(e.currentTarget);
    const clientErrors = validateCourse(input);
    setErrors(clientErrors);
    setFormError("");
    if (Object.keys(clientErrors).length > 0) return;

    setSubmitting(true);
    try {
      await onSubmit(input);
    } catch (err) {
      if (err instanceof ApiError) {
        setErrors(err.fields);
        setFormError(err.message);
      } else {
        setFormError("No pudimos guardar el curso. Inténtalo de nuevo.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} noValidate>
      {formError && (
        <div role="alert" className={styles.alert}>
          {formError}
        </div>
      )}
      <TextField name="title" label="Título" defaultValue={initial?.title} error={shown.title} />
      <TextArea
        name="summary"
        label="Descripción breve"
        rows={2}
        defaultValue={initial?.summary}
        hint="Aparece en la tarjeta del catálogo (20 a 280 caracteres)."
        error={shown.summary}
      />
      <TextArea
        name="description"
        label="Descripción completa"
        rows={6}
        defaultValue={initial?.description}
        hint="Necesaria para publicar (mínimo 50 caracteres)."
        error={shown.description}
      />
      <div className={styles.row}>
        <SelectField
          name="category"
          label="Categoría"
          options={CATEGORIES}
          placeholder="Elige una categoría"
          defaultValue={initial?.category ?? ""}
          error={shown.category}
        />
        <SelectField
          name="level"
          label="Nivel"
          options={LEVELS}
          placeholder="Elige un nivel"
          defaultValue={initial?.level ?? ""}
          error={shown.level}
        />
        <TextField
          name="durationHours"
          label="Duración (horas)"
          type="number"
          min={1}
          max={500}
          inputMode="numeric"
          defaultValue={initial?.durationHours ? String(initial.durationHours) : ""}
          error={shown.durationHours}
        />
      </div>
      <TextArea
        name="objectives"
        label="Objetivos de aprendizaje"
        rows={4}
        defaultValue={initial?.objectives.join("\n")}
        hint="Uno por línea, hasta 10. Al menos uno para publicar."
        error={shown.objectives}
      />
      <TextArea
        name="prerequisites"
        label="Requisitos previos"
        rows={2}
        defaultValue={initial?.prerequisites}
        error={shown.prerequisites}
      />
      <button type="submit" className={styles.button} disabled={submitting}>
        {submitting ? submittingLabel : submitLabel}
      </button>
    </form>
  );
}
