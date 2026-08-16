import React, { useId } from "react";
import styles from "./Field.module.css";
import { cx } from "./cx";

interface FieldWrapperProps {
  label?: string;
  className?: string;
  children: (id: string, controlClass: string) => React.ReactNode;
  mono?: boolean;
}

const FieldWrapper: React.FC<FieldWrapperProps> = ({
  label,
  className,
  mono,
  children,
}) => {
  const id = useId();
  const controlClass = cx(styles.control, mono && styles.mono);
  return (
    <div className={cx(styles.field, className)}>
      {label && (
        <label className={styles.label} htmlFor={id}>
          {label}
        </label>
      )}
      {children(id, controlClass)}
    </div>
  );
};

export interface TextInputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  mono?: boolean;
}

export const TextInput: React.FC<TextInputProps> = ({
  label,
  mono,
  className,
  ...rest
}) => (
  <FieldWrapper label={label} mono={mono} className={className}>
    {(id, controlClass) => <input id={id} className={controlClass} {...rest} />}
  </FieldWrapper>
);

export interface TextAreaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  mono?: boolean;
}

export const TextArea: React.FC<TextAreaProps> = ({
  label,
  mono,
  className,
  ...rest
}) => (
  <FieldWrapper label={label} mono={mono} className={className}>
    {(id, controlClass) => (
      <textarea id={id} className={controlClass} {...rest} />
    )}
  </FieldWrapper>
);

export interface SelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
}

export const Select: React.FC<SelectProps> = ({
  label,
  className,
  children,
  ...rest
}) => (
  <FieldWrapper label={label} className={className}>
    {(id, controlClass) => (
      <select id={id} className={controlClass} {...rest}>
        {children}
      </select>
    )}
  </FieldWrapper>
);
