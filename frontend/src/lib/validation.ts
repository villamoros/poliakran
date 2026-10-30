// Validación en el cliente para dar respuesta inmediata. Replica las reglas
// del backend (internal/identity/domain), que sigue siendo la autoridad.

export const MIN_PASSWORD_LENGTH = 10;
export const MAX_PASSWORD_LENGTH = 128;

export type FieldErrors = Partial<Record<string, string>>;

export interface RegisterValues {
  email: string;
  displayName: string;
  password: string;
}

export interface LoginValues {
  email: string;
  password: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateRegister(v: RegisterValues): FieldErrors {
  const errors: FieldErrors = {};
  const email = v.email.trim();
  const name = v.displayName.trim().replace(/\s+/g, " ");
  const pwLength = [...v.password].length;

  if (!email) errors.email = "Escribe tu correo electrónico.";
  else if (email.length > 254 || !EMAIL_RE.test(email)) errors.email = "Escribe un correo electrónico válido.";

  if (!name) errors.displayName = "Escribe tu nombre.";
  else if ([...name].length < 2 || [...name].length > 80)
    errors.displayName = "El nombre debe tener entre 2 y 80 caracteres.";

  if (pwLength === 0) errors.password = "Escribe una contraseña.";
  else if (pwLength < MIN_PASSWORD_LENGTH)
    errors.password = `La contraseña debe tener al menos ${MIN_PASSWORD_LENGTH} caracteres.`;
  else if (pwLength > MAX_PASSWORD_LENGTH)
    errors.password = `La contraseña no puede superar ${MAX_PASSWORD_LENGTH} caracteres.`;
  else if (!v.password.trim()) errors.password = "La contraseña no puede estar formada solo por espacios.";
  else if (email && v.password.toLowerCase() === email.toLowerCase())
    errors.password = "La contraseña no puede ser igual a tu correo.";

  return errors;
}

export function validateLogin(v: LoginValues): FieldErrors {
  const errors: FieldErrors = {};
  if (!v.email.trim()) errors.email = "Escribe tu correo electrónico.";
  if (!v.password) errors.password = "Escribe tu contraseña.";
  return errors;
}

export interface CourseValues {
  title: string;
  summary: string;
  description: string;
  durationHours: number;
  objectives: string[];
  prerequisites: string;
}

const runes = (s: string) => [...s].length;
const collapse = (s: string) => s.trim().replace(/\s+/g, " ");

// Reglas de internal/catalog/domain.NewDetails: título y descripción breve son
// obligatorios; el resto se exige al publicar y lo comprueba el backend.
export function validateCourse(v: CourseValues): FieldErrors {
  const errors: FieldErrors = {};
  const title = collapse(v.title);
  const summary = collapse(v.summary);

  if (!title) errors.title = "Escribe el título del curso.";
  else if (runes(title) < 5 || runes(title) > 120) errors.title = "El título debe tener entre 5 y 120 caracteres.";

  if (!summary) errors.summary = "Escribe una descripción breve.";
  else if (runes(summary) < 20 || runes(summary) > 280)
    errors.summary = "La descripción breve debe tener entre 20 y 280 caracteres.";

  if (runes(v.description.trim()) > 10000)
    errors.description = "La descripción completa no puede superar 10 000 caracteres.";

  if (!Number.isInteger(v.durationHours) || v.durationHours < 0 || v.durationHours > 500)
    errors.durationHours = "La duración debe estar entre 1 y 500 horas.";

  const objectives = v.objectives.map(collapse).filter(Boolean);
  if (objectives.length > 10) errors.objectives = "Indica como máximo 10 objetivos.";
  else if (objectives.some((o) => runes(o) < 3 || runes(o) > 200))
    errors.objectives = "Cada objetivo debe tener entre 3 y 200 caracteres.";

  if (runes(v.prerequisites.trim()) > 1000)
    errors.prerequisites = "Los requisitos previos no pueden superar 1000 caracteres.";

  return errors;
}

// safeNextPath evita redirecciones abiertas: solo acepta rutas internas.
export function safeNextPath(next: string | null | undefined, fallback = "/panel"): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) return fallback;
  return next;
}
