export function ProgressBar({ percent, label }: { percent: number; label: string }) {
  return (
    <div>
      <div className="flex justify-between text-xs text-gray-600 mb-1">
        <span>{label}</span>
        <span className="font-semibold">{percent}%</span>
      </div>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
        className="h-2 bg-gray-200 rounded-full overflow-hidden"
      >
        <div className="h-full bg-indigo-600 transition-all duration-150" style={{ width: `${percent}%` }} />
      </div>
    </div>
  )
}
