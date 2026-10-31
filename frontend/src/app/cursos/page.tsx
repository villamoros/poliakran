import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { connection } from "next/server";
import { CourseCard } from "@/components/CourseCard";
import { CATEGORIES } from "@/lib/catalog";
import { fetchCatalog } from "@/lib/catalog-server";
import styles from "@/components/ui.module.css";

export const metadata: Metadata = { title: "Catálogo de cursos" };

// LMS-US-003: catálogo público. Solo muestra cursos publicados; el filtrado
// por estado lo hace siempre el backend.
export default function CatalogPage({ searchParams }: PageProps<"/cursos">) {
  return (
    <div className={styles.section}>
      <header className={styles.sectionHeader}>
        <h1 className={styles.title}>Catálogo de cursos</h1>
        <p className={styles.subtitle}>Cursos cortos de competencias digitales y tecnológicas.</p>
      </header>
      <Suspense fallback={<p className={styles.subtitle}>Cargando cursos…</p>}>
        {searchParams.then((sp) => (
          <CourseList category={typeof sp.categoria === "string" ? sp.categoria : ""} />
        ))}
      </Suspense>
    </div>
  );
}

async function CourseList({ category }: { category: string }) {
  await connection();
  const selected = category in CATEGORIES ? category : "";
  let page;
  try {
    page = await fetchCatalog(selected);
  } catch {
    return (
      <div role="alert" className={styles.alert}>
        No pudimos cargar el catálogo. Inténtalo de nuevo en unos minutos.
      </div>
    );
  }

  return (
    <>
      <nav aria-label="Categorías" className={styles.filters}>
        <Link href="/cursos" className={styles.chip} aria-current={selected === "" ? "page" : undefined}>
          Todas
        </Link>
        {Object.entries(CATEGORIES).map(([value, label]) => (
          <Link
            key={value}
            href={`/cursos?categoria=${value}`}
            className={styles.chip}
            aria-current={selected === value ? "page" : undefined}
          >
            {label}
          </Link>
        ))}
      </nav>
      {page.items.length === 0 ? (
        <p className={styles.empty}>
          {selected ? "Todavía no hay cursos publicados en esta categoría." : "Todavía no hay cursos publicados."}
        </p>
      ) : (
        <div className={styles.grid}>
          {page.items.map((course) => (
            <CourseCard key={course.slug} course={course} />
          ))}
        </div>
      )}
    </>
  );
}
