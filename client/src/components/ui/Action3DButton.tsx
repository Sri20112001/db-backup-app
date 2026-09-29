import React from 'react';

interface Action3DButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children: React.ReactNode;
}

// Updated button style: a highly polished, glowing/tactile button 
// that perfectly matches the Physical Workbench / Cyberdeck Night themes.
const Action3DButton: React.FC<Action3DButtonProps> = ({ children, className = '', ...props }) => {
  return (
    <button
      className={`
        relative inline-flex items-center justify-center cursor-pointer outline-none align-middle
        text-[13px] font-semibold text-white dark:text-[#251a10]
        px-4 h-9 rounded-lg
        bg-primary
        overflow-hidden
        transition-all duration-300 ease-out
        hover:scale-[1.02] hover:-translate-y-0.5
        active:scale-[0.98] active:translate-y-0
        disabled:opacity-50 disabled:cursor-not-allowed disabled:pointer-events-none disabled:transform-none
        
        shadow-[0_4px_12px_rgba(234,88,12,0.25)]
        hover:shadow-[0_8px_20px_rgba(234,88,12,0.4)]
        dark:shadow-[0_0_15px_rgba(245,158,11,0.3)]
        dark:hover:shadow-[0_0_25px_rgba(245,158,11,0.5)]

        before:content-[''] before:absolute before:inset-0
        before:rounded-lg
        before:bg-gradient-to-b before:from-white/20 before:to-transparent
        before:pointer-events-none

        ${className}
      `}
      {...props}
    >
      <span className="relative z-10 flex items-center gap-2 drop-shadow-sm">{children}</span>
    </button>
  );
};

export default Action3DButton;
