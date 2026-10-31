import type { CatalogPage, PublicCourse } from "./catalog";

// Lectura del catálogo público desde Server Components. Se habla directamente
// con el backend (sin pasar por el rewrite de /api) porque no hay navegador.
const apiUrl = process.env.API_URL ?? "http://localhost:8080";

export class CatalogUnavailableError extends Error {}

async function get(path: string): Promise<Response> {
  try {
    return await fetch(`${apiUrl}${path}`, {
      headers: { Accept: "application/json" },
    });
  } catch {
    throw new CatalogUnavailableError("El catálogo no está disponible en este momento.");
  }
}

export async function fetchCatalog(category?: string): Promise<CatalogPage> {
  const params = new URLSearchParams({ limit: "100" });
  if (category) params.set("category", category);
  const res = await get(`/api/v1/catalog/courses?${params}`);
  if (!res.ok) throw new CatalogUnavailableError(`El catálogo respondió ${res.status}.`);
  return res.json();
}

/** Devuelve null si el curso no existe o no está publicado. */
export async function fetchPublishedCourse(slug: string): Promise<PublicCourse | null> {
  const res = await get(`/api/v1/catalog/courses/${encodeURIComponent(slug)}`);
  if (res.status === 404) return null;
  if (!res.ok) throw new CatalogUnavailableError(`El catálogo respondió ${res.status}.`);
  const data = await res.json();
  return data.course;
}
