import React from 'react';

interface NeoToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  id?: string;
}

const NeoToggle: React.FC<NeoToggleProps> = ({ checked, onChange, id = 'neo-toggle' }) => {
  return (
    <div
      className="relative inline-flex flex-col font-sans select-none"
      style={{
        '--toggle-width': '80px',
        '--toggle-height': '38px',
        '--toggle-bg': '#f9f9ff',
        '--toggle-off-color': '#737686',
        '--toggle-on-color': '#2563eb',
        '--toggle-transition': '0.4s cubic-bezier(0.25, 1, 0.5, 1)',
      } as React.CSSProperties}
    >
      <input
        className="peer absolute opacity-0 w-0 h-0"
        id={id}
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      <label
        className="relative block cursor-pointer [transform:translateZ(0)] [perspective:500px] w-[var(--toggle-width)] h-[var(--toggle-height)]"
        htmlFor={id}
      >
        <div className="absolute inset-0 rounded-[calc(var(--toggle-height)/2)] overflow-hidden [transform-style:preserve-3d] [transform:translateZ(-1px)] transition-transform duration-500 ease-out shadow-[0_2px_10px_rgba(0,0,0,0.1),inset_0_0_0_1px_rgba(0,0,0,0.05)]">
          <div className="absolute inset-0 bg-[var(--toggle-bg)] opacity-100 transition-all duration-500 ease-out" />
          
          <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(115,118,134,0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgba(115,118,134,0.05)_1px,transparent_1px)] [background-size:5px_5px] opacity-0 transition-opacity duration-500 ease-out peer-checked:opacity-100" />
          
          <div className="absolute inset-[1px] rounded-[calc(var(--toggle-height)/2)] bg-[linear-gradient(90deg,transparent,rgba(37,99,235,0.2))] opacity-0 transition-all duration-500 ease-out peer-checked:opacity-100" />
        </div>

        <div className="absolute bottom-[6px] right-[10px] h-[10px] flex items-end gap-[2px] opacity-0 transition-opacity duration-500 ease-out peer-checked:opacity-100">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className={`w-[2px] h-[3px] bg-[var(--toggle-on-color)] opacity-80 animate-pulse delay-${i * 100}`} />
          ))}
        </div>

        <div className="absolute top-[4px] left-[4px] w-[30px] h-[30px] rounded-full [transform-style:preserve-3d] transition-transform duration-500 ease-out z-10 peer-checked:translate-x-[calc(var(--toggle-width)-38px)]">
          <div className="absolute inset-0 rounded-full border border-black/5 bg-[var(--toggle-off-color)] shadow-[0_2px_10px_rgba(0,0,0,0.1)] transition-all duration-500 ease-out peer-checked:bg-[var(--toggle-on-color)] peer-checked:border-[rgba(37,99,235,0.3)] peer-checked:shadow-[0_0_15px_rgba(37,99,235,0.5)]" />
          <div className="absolute inset-[5px] rounded-full bg-[linear-gradient(135deg,rgba(255,255,255,0.5),transparent)] transition-all duration-500 ease-out overflow-hidden flex items-center justify-center">
            <div className="relative w-[10px] h-[10px] transition-all duration-500 ease-out">
              <div className="absolute top-1/2 left-1/2 w-[10px] h-[2px] bg-white -translate-x-1/2 -translate-y-1/2 transition-all duration-500 ease-out peer-checked:h-[8px] peer-checked:w-[8px] peer-checked:rounded-full peer-checked:bg-transparent peer-checked:border peer-checked:border-white" />
              <div className="absolute inset-0 rounded-full border border-white scale-0 opacity-0 transition-all duration-500 ease-out peer-checked:scale-125 peer-checked:opacity-30 peer-checked:animate-ping" />
            </div>
          </div>
        </div>
        
        <div className="absolute -bottom-[20px] left-0 w-full flex justify-center">
          <div className="flex items-center gap-1">
            <div className="w-[6px] h-[6px] rounded-full bg-[var(--toggle-off-color)] transition-all duration-500 ease-out peer-checked:bg-[var(--toggle-on-color)] peer-checked:shadow-[0_0_8px_var(--toggle-on-color)]" />
            <div className="text-[9px] font-semibold text-[var(--toggle-off-color)] tracking-[1px] transition-all duration-500 ease-out peer-checked:text-[var(--toggle-on-color)]">
              {checked ? 'ACTIVE' : 'STANDBY'}
            </div>
          </div>
        </div>
      </label>
    </div>
  );
};

export default NeoToggle;
