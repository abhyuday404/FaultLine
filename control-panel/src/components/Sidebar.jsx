import React from 'react';

const items = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'rules', label: 'Rules'},
  { key: 'analysis', label: 'Analysis'},
];

export default function Sidebar({ active, onNavigate }) {
  return (
    <aside className="sidebar">
      <div className="brand">FaultLine</div>
      <nav>
        {items.map(it => (
          <button
            key={it.key}
            className={`nav-item ${active === it.key ? 'active' : ''}`}
            onClick={() => onNavigate(it.key)}
          >
            <span className="icon" aria-hidden>{it.icon}</span>
            <span>{it.label}</span>
          </button>
        ))}
      </nav>
      <div className="sidebar-footer">
        <span className="muted">v1.0</span>
      </div>
    </aside>
  );
}
