// src/components/ui/cover.jsx
import React from "react";

export function Cover({ children }) {
  return (
    <span className="bg-gradient-to-r from-purple-500 to-indigo-500 text-transparent bg-clip-text">
      {children}
    </span>
  );
}
