// Vocabulario y tipos del catálogo, compartidos por las vistas públicas y las
// de administración. Los valores replican internal/catalog/domain.

export const CATEGORIES: Record<string, string> = {
  programacion: "Programación",
  "bases-de-datos": "Bases de datos",
  "desarrollo-web": "Desarrollo web",
  "herramientas-software": "Herramientas de software",
  otras: "Otras competencias digitales",
};

export const LEVELS: Record<string, string> = {
  basico: "Básico",
  intermedio: "Intermedio",
  avanzado: "Avanzado",
};

export const STATUS_LABELS: Record<CourseStatus, string> = {
  draft: "Borrador",
  published: "Publicado",
  withdrawn: "Retirado",
};

export type CourseStatus = "draft" | "published" | "withdrawn";

/** Curso tal como lo ve un visitante en el catálogo público. */
export interface PublicCourse {
  slug: string;
  title: string;
  summary: string;
  description?: string;
  category: string;
  level: string;
  durationHours: number;
  objectives?: string[];
  prerequisites?: string;
  publishedAt: string | null;
}

export interface CatalogPage {
  items: PublicCourse[];
  total: number;
  limit: number;
  offset: number;
}

/** Datos editables de un curso (cuerpo de creación y edición). */
export interface CourseInput {
  title: string;
  summary: string;
  description: string;
  category: string;
  level: string;
  durationHours: number;
  objectives: string[];
  prerequisites: string;
}

/** Curso completo para administración. */
export interface AdminCourse extends CourseInput {
  id: string;
  slug: string;
  status: CourseStatus;
  publishedAt: string | null;
  withdrawnAt: string | null;
  createdAt: string;
  updatedAt: string;
  version: number;
}

export function categoryLabel(value: string): string {
  return CATEGORIES[value] ?? "Sin categoría";
}

export function levelLabel(value: string): string {
  return LEVELS[value] ?? "Sin nivel";
}

export function durationLabel(hours: number): string {
  if (hours <= 0) return "Duración por definir";
  return hours === 1 ? "1 hora" : `${hours} horas`;
}
