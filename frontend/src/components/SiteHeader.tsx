import Link from "next/link";
import styles from "./ui.module.css";

export function SiteHeader() {
  return (
    <header className={styles.header}>
      <Link href="/" className={styles.brand}>
        Politécnico <span>Akran</span>
      </Link>
    </header>
  );
}
