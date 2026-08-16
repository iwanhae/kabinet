import React from "react";

export interface SparklineProps {
  /** Primary series (rendered as a filled steel-blue area). */
  points: number[];
  /** Optional overlay line (signal red), same scale as points. */
  warnPoints?: number[];
  width?: number;
  height?: number;
}

const toPath = (
  values: number[],
  max: number,
  width: number,
  height: number,
): string => {
  const step = values.length > 1 ? width / (values.length - 1) : 0;
  return values
    .map((v, i) => {
      const x = (i * step).toFixed(1);
      const y = (height - 1 - (v / max) * (height - 2)).toFixed(1);
      return `${i === 0 ? "M" : "L"}${x},${y}`;
    })
    .join(" ");
};

/** Dependency-free inline-SVG sparkline for table rows. */
export const Sparkline: React.FC<SparklineProps> = ({
  points,
  warnPoints,
  width = 120,
  height = 28,
}) => {
  if (points.length === 0) return null;
  const max = Math.max(...points, ...(warnPoints ?? []), 1);
  const areaPath = `${toPath(points, max, width, height)} L${width},${height} L0,${height} Z`;

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      aria-hidden="true"
      style={{ display: "block" }}
    >
      <path d={areaPath} fill="var(--steel-soft)" />
      <path
        d={toPath(points, max, width, height)}
        fill="none"
        stroke="var(--steel)"
        strokeWidth={1.5}
      />
      {warnPoints && warnPoints.some((v) => v > 0) && (
        <path
          d={toPath(warnPoints, max, width, height)}
          fill="none"
          stroke="var(--signal)"
          strokeWidth={1.5}
        />
      )}
    </svg>
  );
};
