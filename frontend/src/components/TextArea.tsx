import styles from "./ui.module.css";

interface Props extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  name: string;
  label: string;
  error?: string;
  hint?: string;
}

export function TextArea({ name, label, error, hint, ...textarea }: Props) {
  const describedBy = [error && `${name}-error`, hint && `${name}-hint`].filter(Boolean).join(" ") || undefined;
  return (
    <div className={styles.field}>
      <label htmlFor={name} className={styles.label}>
        {label}
      </label>
      <textarea
        id={name}
        name={name}
        className={`${styles.input} ${styles.textarea}`}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        {...textarea}
      />
      {hint && !error && (
        <span id={`${name}-hint`} className={styles.hint}>
          {hint}
        </span>
      )}
      {error && (
        <span id={`${name}-error`} className={styles.fieldError}>
          {error}
        </span>
      )}
    </div>
  );
}
