import React from 'react';

interface Action3DButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children: React.ReactNode;
}

const Action3DButton: React.FC<Action3DButtonProps> = ({ children, className = '', ...props }) => {
  return (
    <button
      className={`
        inline-flex items-center justify-center cursor-pointer outline-none align-middle
        text-[13px] font-medium text-white
        px-4 h-9 rounded-lg
        bg-[#2563eb]
        border border-[#1d4ed8]
        shadow-sm
        transition-all duration-150 ease-out
        
        hover:bg-[#1d4ed8]
        hover:shadow-md
        
        active:bg-[#1e40af]
        active:scale-[0.98]
        active:shadow-sm

        disabled:opacity-60 disabled:cursor-not-allowed disabled:pointer-events-none

        ${className}
      `}
      {...props}
    >
      <span className="flex items-center gap-2">{children}</span>
    </button>
  );
};

export default Action3DButton;
