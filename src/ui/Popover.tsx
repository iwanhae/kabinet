import React, { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import styles from "./Popover.module.css";

export interface PopoverProps {
  open: boolean;
  anchorEl: HTMLElement | null;
  onClose: () => void;
  /** Horizontal alignment relative to the anchor. */
  align?: "start" | "end";
  children: React.ReactNode;
}

export const Popover: React.FC<PopoverProps> = ({
  open,
  anchorEl,
  onClose,
  align = "start",
  children,
}) => {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ top: 0, left: 0 });

  useLayoutEffect(() => {
    if (!open || !anchorEl) return;

    const place = () => {
      const anchor = anchorEl.getBoundingClientRect();
      const el = ref.current;
      const width = el?.offsetWidth ?? 0;
      const height = el?.offsetHeight ?? 0;

      let left = align === "end" ? anchor.right - width : anchor.left;
      left = Math.max(8, Math.min(left, window.innerWidth - width - 8));

      let top = anchor.bottom + 4;
      if (top + height > window.innerHeight - 8) {
        top = Math.max(8, anchor.top - height - 4);
      }
      setPos({ top, left });
    };

    place();
    window.addEventListener("resize", place);
    window.addEventListener("scroll", place, true);
    return () => {
      window.removeEventListener("resize", place);
      window.removeEventListener("scroll", place, true);
    };
  }, [open, anchorEl, align]);

  useEffect(() => {
    if (!open) return;

    const onMouseDown = (e: MouseEvent) => {
      const target = e.target as Node;
      if (ref.current?.contains(target)) return;
      if (anchorEl?.contains(target)) return;
      onClose();
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };

    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, anchorEl, onClose]);

  if (!open) return null;

  return createPortal(
    <div
      ref={ref}
      className={styles.popover}
      style={{ top: pos.top, left: pos.left }}
      role="dialog"
    >
      {children}
    </div>,
    document.body,
  );
};
