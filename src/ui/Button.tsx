import React from "react";
import styles from "./Button.module.css";
import { cx } from "./cx";

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "solid" | "outline" | "ghost";
  size?: "sm" | "md";
  ref?: React.Ref<HTMLButtonElement>;
}

export const Button: React.FC<ButtonProps> = ({
  variant = "outline",
  size = "md",
  className,
  type = "button",
  ...rest
}) => (
  <button
    type={type}
    className={cx(styles.button, styles[variant], styles[size], className)}
    {...rest}
  />
);

export interface IconButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  size?: "sm" | "md";
  label: string;
}

export const IconButton: React.FC<IconButtonProps> = ({
  size = "md",
  label,
  className,
  type = "button",
  ...rest
}) => (
  <button
    type={type}
    aria-label={label}
    title={label}
    className={cx(
      styles.button,
      styles.ghost,
      styles[size],
      styles.iconOnly,
      className,
    )}
    {...rest}
  />
);
