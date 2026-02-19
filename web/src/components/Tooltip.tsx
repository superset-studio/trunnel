import type { ReactNode } from 'react';

interface TooltipProps {
  content: ReactNode;
  children: ReactNode;
  wide?: boolean;
}

export function Tooltip({ content, children, wide }: TooltipProps) {
  return (
    <span className="group relative inline-block">
      {children}
      <span
        className={`invisible group-hover:visible opacity-0 group-hover:opacity-100 transition-opacity absolute bottom-full left-1/2 -translate-x-1/2 mb-2 bg-slate-800 text-white text-xs rounded py-2 px-3 z-50 ${wide ? 'w-72 whitespace-normal' : 'whitespace-nowrap'}`}
      >
        {content}
        <span className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-800" />
      </span>
    </span>
  );
}

export function InfoIcon({ className = '' }: { className?: string }) {
  return (
    <svg
      className={`inline-block h-4 w-4 text-slate-400 hover:text-slate-600 transition-colors ${className}`}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <circle cx="12" cy="12" r="10" />
      <path strokeLinecap="round" d="M12 16v-4m0-4h.01" />
    </svg>
  );
}
