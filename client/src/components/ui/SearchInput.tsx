import React, { useEffect, useRef } from 'react';
import { Search } from 'lucide-react';

type SearchInputProps = React.InputHTMLAttributes<HTMLInputElement>

const SearchInput: React.FC<SearchInputProps> = ({ className = '', ...props }) => {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <div className={`relative inline-flex items-center w-full ${className}`}>
      <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
      <input
        ref={inputRef}
        type="text"
        className="w-full h-9 pl-9 pr-14 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] placeholder:text-on-surface-variant outline-none focus:border-primary transition-colors"
        {...props}
      />
      <kbd className="pointer-events-none absolute right-[0.3rem] top-[0.3rem] flex h-[22px] select-none items-center gap-1 rounded border border-surface-variant bg-surface-container-low px-1.5 font-mono text-[10px] font-medium text-on-surface-variant opacity-100">
        <span className="text-xs">⌘</span>K
      </kbd>
    </div>
  );
};

export default SearchInput;
