import React, { useEffect, useRef, useState } from "react";
import dayjs from "dayjs";
import { ChevronDown, Clock, RefreshCw } from "lucide-react";
import { useTimeRange } from "../hooks/useUrlParams";
import { isRelativeTime, parseTime } from "../utils/timeRange";
import { Button, IconButton, Popover, TextInput } from "../ui";
import styles from "./TimeRangePicker.module.css";

const QUICK_RANGES = [
  { label: "Last 5 minutes", from: "now-5m" },
  { label: "Last 15 minutes", from: "now-15m" },
  { label: "Last 30 minutes", from: "now-30m" },
  { label: "Last 1 hour", from: "now-1h" },
  { label: "Last 3 hours", from: "now-3h" },
  { label: "Last 6 hours", from: "now-6h" },
  { label: "Last 12 hours", from: "now-12h" },
  { label: "Last 24 hours", from: "now-24h" },
  { label: "Last 2 days", from: "now-2d" },
  { label: "Last 7 days", from: "now-7d" },
  { label: "Last 30 days", from: "now-30d" },
];

export const TimeRangePicker: React.FC = () => {
  const { rawFrom, rawTo, setTimeRange, refreshTimeRange } = useTimeRange();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [draftFrom, setDraftFrom] = useState(rawFrom);
  const [draftTo, setDraftTo] = useState(rawTo);

  useEffect(() => {
    setDraftFrom(rawFrom);
    setDraftTo(rawTo);
  }, [rawFrom, rawTo, open]);

  const quickMatch = QUICK_RANGES.find(
    (qr) => qr.from === rawFrom && rawTo === "now",
  );
  const isRelative = isRelativeTime(rawFrom) || isRelativeTime(rawTo);

  const label = quickMatch
    ? quickMatch.label
    : isRelative
      ? `${rawFrom} → ${rawTo}`
      : `${dayjs(rawFrom).format("MMM D, HH:mm")} → ${dayjs(rawTo).format("MMM D, HH:mm")}`;

  const draftValid =
    parseTime(draftFrom).isValid() &&
    parseTime(draftTo).isValid() &&
    parseTime(draftTo).diff(parseTime(draftFrom)) > 0;

  const apply = (from: string, to: string) => {
    setTimeRange(from, to);
    setOpen(false);
  };

  /** Feeds a native datetime-local pick back into the raw text draft. */
  const fromPicker = (value: string, set: (v: string) => void) => {
    if (value) set(dayjs(value).format("YYYY-MM-DDTHH:mm:ssZ"));
  };

  return (
    <>
      <Button
        ref={triggerRef}
        variant="outline"
        className={styles.trigger}
        onClick={() => setOpen((v) => !v)}
      >
        <Clock size={14} />
        {label}
        <ChevronDown size={14} />
      </Button>
      {isRelative && (
        <IconButton label="Refresh time range" onClick={refreshTimeRange}>
          <RefreshCw size={15} />
        </IconButton>
      )}

      <Popover
        open={open}
        anchorEl={triggerRef.current}
        onClose={() => setOpen(false)}
        align="end"
      >
        <div className={styles.panel}>
          <div className={styles.quickList}>
            {QUICK_RANGES.map((qr) => (
              <button
                key={qr.from}
                type="button"
                className={`${styles.quickItem} ${
                  quickMatch?.from === qr.from ? styles.quickItemActive : ""
                }`}
                onClick={() => apply(qr.from, "now")}
              >
                {qr.label}
              </button>
            ))}
          </div>

          <div className={styles.custom}>
            <div className="eyebrow">Custom range</div>
            <div className={styles.pickerRow}>
              <TextInput
                label="From"
                mono
                value={draftFrom}
                onChange={(e) => setDraftFrom(e.target.value)}
                placeholder="now-30m or ISO time"
              />
              <input
                type="datetime-local"
                className={styles.nativePicker}
                aria-label="Pick start time"
                onChange={(e) => fromPicker(e.target.value, setDraftFrom)}
              />
            </div>
            <div className={styles.pickerRow}>
              <TextInput
                label="To"
                mono
                value={draftTo}
                onChange={(e) => setDraftTo(e.target.value)}
                placeholder="now or ISO time"
              />
              <input
                type="datetime-local"
                className={styles.nativePicker}
                aria-label="Pick end time"
                onChange={(e) => fromPicker(e.target.value, setDraftTo)}
              />
            </div>
            <div className={styles.hint}>
              now-30s · now-15m · now-2h · now-7d · now-1w
            </div>
            {!draftValid && (
              <div className={styles.invalid}>
                Enter a valid range (from must be before to).
              </div>
            )}
            <div className={styles.footer}>
              <Button variant="ghost" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button
                variant="solid"
                disabled={!draftValid}
                onClick={() => apply(draftFrom.trim(), draftTo.trim())}
              >
                Apply range
              </Button>
            </div>
          </div>
        </div>
      </Popover>
    </>
  );
};
