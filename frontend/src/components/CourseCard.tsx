import Link from "next/link";
import { categoryLabel, durationLabel, levelLabel, type PublicCourse } from "@/lib/catalog";
import styles from "./ui.module.css";

export function CourseCard({ course }: { course: PublicCourse }) {
  return (
    <article className={styles.courseCard}>
      <span className={styles.badge}>{categoryLabel(course.category)}</span>
      <h2 className={styles.courseTitle}>
        <Link href={`/cursos/${course.slug}`}>{course.title}</Link>
      </h2>
      <p className={styles.courseSummary}>{course.summary}</p>
      <p className={styles.meta}>
        {levelLabel(course.level)} · {durationLabel(course.durationHours)}
      </p>
    </article>
  );
}
