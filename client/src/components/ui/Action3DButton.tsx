import React from 'react';

interface Action3DButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children: React.ReactNode;
}

// Uiverse-style 3D press button in application colors (no styled-components).
// The raised look comes from a ::before plate (edge color + white base layer)
// sitting 0.75em below the face; hover/active translate the face down into
// it. Sizes/typography stay overridable via className (! utilities win).
const Action3DButton: React.FC<Action3DButtonProps> = ({ children, className = '', ...props }) => {
  return (
    <button
      className={`
        relative inline-flex items-center justify-center cursor-pointer outline-none align-middle
        text-[13px] font-medium text-on-primary
        px-4 h-9 rounded-lg
        bg-primary
        [transform-style:preserve-3d]
        transition-[transform,background-color] duration-150 ease-out
        hover:bg-primary-container hover:[transform:translate(0,0.25em)]
        active:bg-[#1d4ed8] active:[transform:translate(0,0.75em)]
        disabled:opacity-60 disabled:cursor-not-allowed disabled:pointer-events-none
        disabled:[transform:none]

        before:content-[''] before:absolute before:inset-0
        before:rounded-[inherit]
        before:bg-[#1d4ed8]
        before:[box-shadow:0_0_0_2px_#1d4ed8,0_0.625em_0_0_#ffffff]
        before:[transform:translate3d(0,0.75em,-1em)]
        before:transition-all before:duration-150 before:ease-out
        hover:before:[box-shadow:0_0_0_2px_#1d4ed8,0_0.5em_0_0_#ffffff]
        hover:before:[transform:translate3d(0,0.5em,-1em)]
        active:before:[box-shadow:0_0_0_2px_#1d4ed8,0_0_#ffffff]
        active:before:[transform:translate3d(0,0,-1em)]

        ${className}
      `}
      {...props}
    >
      <span className="flex items-center gap-2 [transform:translateZ(1px)]">{children}</span>
    </button>
  );
};

export default Action3DButton;
