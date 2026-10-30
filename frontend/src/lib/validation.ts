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

// safeNextPath evita redirecciones abiertas: solo acepta rutas internas.
export function safeNextPath(next: string | null | undefined, fallback = "/panel"): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) return fallback;
  return next;
}
