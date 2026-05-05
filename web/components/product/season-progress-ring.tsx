import styles from "./season-progress-ring.module.css";

export interface SeasonProgressRingProps {
  day: number; // 1..7
  daysLeft: number;
  status: "active" | "completed";
}

const SIZE = 64;
const STROKE_WIDTH = 6;
const RADIUS = 26;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS; // ≈ 163.36

export function SeasonProgressRing({ day, daysLeft, status }: SeasonProgressRingProps) {
  const fraction = Math.min(Math.max(day / 7, 0), 1);
  const dashOffset = CIRCUMFERENCE * (1 - fraction);
  const isCompleted = status === "completed";
  const arcColor = isCompleted ? "#555" : "#E5FF00";
  const labelColor = isCompleted ? "#555" : "#E5FF00";
  const centerText = isCompleted ? "✓" : String(day);

  return (
    <div className={styles.wrapper} title={`День ${day} из 7 · осталось ${daysLeft} дн.`}>
      <svg
        width={SIZE}
        height={SIZE}
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        className={styles.svg}
        aria-hidden="true"
      >
        {/* Track */}
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={RADIUS}
          fill="none"
          stroke="#2A2A2A"
          strokeWidth={STROKE_WIDTH}
        />
        {/* Progress arc */}
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={RADIUS}
          fill="none"
          stroke={arcColor}
          strokeWidth={STROKE_WIDTH}
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={dashOffset}
          strokeLinecap="round"
          transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}
          className={styles.arc}
        />
        {/* Center label */}
        <text
          x={SIZE / 2}
          y={SIZE / 2}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize="14"
          fontWeight="700"
          fill={labelColor}
        >
          {centerText}
        </text>
      </svg>
    </div>
  );
}
