import React from 'react';

export default function StatCard({ label, value, trend }) {
  return (
    <div className="stat-card">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
      {trend && <div className={`stat-trend ${trend}`}>{trend === 'up' ? '▲' : '▼'} {Math.floor(Math.random()*10)+1}%</div>}
    </div>
  );
}
