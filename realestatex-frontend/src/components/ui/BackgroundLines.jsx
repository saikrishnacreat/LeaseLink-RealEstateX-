import React from "react"

export function BackgroundLines({ className = '', children }) {
  return (
    <div className={`relative overflow-hidden ${className}`}>
      <div className="absolute inset-0 bg-gradient-to-r from-purple-300/30 to-indigo-300/30 backdrop-blur-sm z-0" />
      <div className="relative z-10">
        {children}
      </div>
    </div>
  )
}
