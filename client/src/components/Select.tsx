import React, { useState, useRef, useEffect } from "react";
import { Search, ChevronDown, Check, RotateCcw, Database } from "lucide-react";

interface DatabaseSelectProps {
  dbs: string[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export const Select: React.FC<DatabaseSelectProps> = ({
  dbs,
  value,
  onChange,
  placeholder = "Select a database…",
  className = "",
}) => {
  // If the initial value isn't in `dbs` and is non-empty, default to manual mode
  const [isManual, setIsManual] = useState<boolean>(
    Boolean(value && !dbs.includes(value))
  );
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState<number>(-1);

  const containerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const manualInputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  // Filter databases based on search input
  const filteredDbs = dbs.filter((db) =>
    db.toLowerCase().includes(searchQuery.trim().toLowerCase())
  );

  // Total selectable items = filtered databases + 1 ("Type manually" action)
  const manualOptionIndex = filteredDbs.length;

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
        setSearchQuery("");
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen) {
      setHighlightedIndex(-1);
      setTimeout(() => searchInputRef.current?.focus(), 50);
    }
  }, [isOpen]);

  // Focus manual input when entering manual mode
  useEffect(() => {
    if (isManual) {
      setTimeout(() => manualInputRef.current?.focus(), 50);
    }
  }, [isManual]);

  // Handle keyboard navigation inside the dropdown
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen) {
      if (e.key === "Enter" || e.key === "ArrowDown" || e.key === " ") {
        e.preventDefault();
        setIsOpen(true);
      }
      return;
    }

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setHighlightedIndex((prev) =>
          prev < manualOptionIndex ? prev + 1 : 0
        );
        break;
      case "ArrowUp":
        e.preventDefault();
        setHighlightedIndex((prev) =>
          prev > 0 ? prev - 1 : manualOptionIndex
        );
        break;
      case "Enter":
        e.preventDefault();
        if (highlightedIndex === manualOptionIndex) {
          handleSelectManual();
        } else if (highlightedIndex >= 0 && highlightedIndex < filteredDbs.length) {
          handleSelect(filteredDbs[highlightedIndex]);
        }
        break;
      case "Escape":
        e.preventDefault();
        setIsOpen(false);
        break;
      case "Tab":
        setIsOpen(false);
        break;
    }
  };

  const handleSelect = (dbName: string) => {
    onChange(dbName);
    setIsOpen(false);
    setSearchQuery("");
  };

  const handleSelectManual = () => {
    setIsManual(true);
    setIsOpen(false);
    setSearchQuery("");
  };

  const handleSwitchBackToSelect = () => {
    setIsManual(false);
    // If the value was custom-typed and not in `dbs`, reset it
    if (!dbs.includes(value)) {
      onChange("");
    }
  };

  // --- MANUAL INPUT VIEW ---
  if (isManual) {
    return (
      <div className={`relative flex items-center ${className}`}>
        <div className="absolute left-3 text-outline pointer-events-none flex items-center">
          <Database size={15} />
        </div>
        <input
          ref={manualInputRef}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Enter database name manually…"
          className="w-full h-10 pl-9 pr-24 rounded-lg bg-surface-container-lowest border border-primary text-[13px] text-on-surface placeholder-[#737686] outline-none shadow-sm focus:ring-2 focus:ring-blue-500/20 font-mono transition-all"
        />
        <button
          type="button"
          onClick={handleSwitchBackToSelect}
          className="absolute right-2 px-2 py-1 text-[11px] font-medium text-on-primary-container bg-primary-container/50 hover:bg-primary-container rounded flex items-center gap-1 transition-colors"
          title="Switch back to database list"
        >
          <RotateCcw size={12} />
          Choose list
        </button>
      </div>
    );
  }

  // --- SEARCHABLE DROPDOWN VIEW ---
  return (
    <div
      ref={containerRef}
      onKeyDown={handleKeyDown}
      className={`relative w-full ${className}`}
    >
      {/* Trigger Button */}
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        onClick={() => setIsOpen((prev) => !prev)}
        className="w-full h-10 px-3.5 rounded-lg bg-surface-container-lowest border border-surface-variant hover:border-[#b8c2ea] focus:border-primary focus:ring-2 focus:ring-blue-500/20 shadow-sm flex items-center justify-between text-left transition-all outline-none"
      >
        <span
          className={`text-[13px] truncate ${
            value ? "text-on-surface font-medium" : "text-outline"
          }`}
        >
          {value || placeholder}
        </span>
        <ChevronDown
          size={16}
          className={`text-outline shrink-0 transition-transform duration-200 ${
            isOpen ? "rotate-180" : ""
          }`}
        />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute z-50 left-0 right-0 top-full mt-1.5 bg-surface-container-lowest border border-surface-variant rounded-xl shadow-lg shadow-black/5 overflow-hidden animate-in fade-in-0 zoom-in-95 duration-100">
          {/* Search Box */}
          <div className="p-2 border-b border-surface-variant bg-slate-50/50">
            <div className="relative flex items-center">
              <Search
                size={14}
                className="absolute left-2.5 text-outline pointer-events-none"
              />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setHighlightedIndex(0);
                }}
                placeholder="Search databases…"
                className="w-full h-8 pl-8 pr-3 text-[12px] bg-surface-container-lowest border border-surface-variant rounded-md text-on-surface placeholder-[#737686] outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-all"
              />
            </div>
          </div>

          {/* Items List */}
          <ul
            ref={listRef}
            role="listbox"
            className="max-h-42 overflow-y-auto p-1.5 space-y-0.5"
          >
            {filteredDbs.length > 0 ? (
              filteredDbs.map((db, idx) => {
                const isSelected = value === db;
                const isHighlighted = highlightedIndex === idx;

                return (
                  <li
                    key={db}
                    role="option"
                    aria-selected={isSelected}
                    onClick={() => handleSelect(db)}
                    onMouseEnter={() => setHighlightedIndex(idx)}
                    className={`flex items-center justify-between px-2.5 py-1.5 rounded-lg text-[13px] cursor-pointer font-mono transition-colors ${
                      isSelected
                        ? "bg-surface-container-high text-primary font-semibold"
                        : isHighlighted
                        ? "bg-slate-100 text-on-surface"
                        : "text-on-surface hover:bg-slate-50"
                    }`}
                  >
                    <span className="truncate">{db}</span>
                    {isSelected && (
                      <Check size={14} className="text-primary shrink-0" />
                    )}
                  </li>
                );
              })
            ) : (
              <li className="px-3 py-3 text-center text-[12px] text-outline">
                No matching databases found.
              </li>
            )}

           
          </ul>
        </div>
      )}
    </div>
  );
};