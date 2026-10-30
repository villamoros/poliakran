import styles from "./ui.module.css";

interface Props extends React.SelectHTMLAttributes<HTMLSelectElement> {
  name: string;
  label: string;
  options: Record<string, string>;
  placeholder: string;
  error?: string;
}

export function SelectField({ name, label, options, placeholder, error, ...select }: Props) {
  return (
    <div className={styles.field}>
      <label htmlFor={name} className={styles.label}>
        {label}
      </label>
      <select
        id={name}
        name={name}
        className={styles.input}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${name}-error` : undefined}
        {...select}
      >
        <option value="">{placeholder}</option>
        {Object.entries(options).map(([value, text]) => (
          <option key={value} value={value}>
            {text}
          </option>
        ))}
      </select>
      {error && (
        <span id={`${name}-error`} className={styles.fieldError}>
          {error}
        </span>
      )}
    </div>
  );
}
