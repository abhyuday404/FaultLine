import React from 'react';

export default function Header({ title, subtitle }) {
  return (
    <header className="topbar">
      <div>
        <h1 className="title">{title}</h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
      <div className="topbar-actions">
        <button className="btn ghost" onClick={() => window.location.reload()}>↻ Refresh</button>
        <a className="btn ghost" href="https://github.com/abhyuday404/FaultLine" target="_blank" rel="noreferrer">GitHub</a>
      </div>
    </header>
  );
}
