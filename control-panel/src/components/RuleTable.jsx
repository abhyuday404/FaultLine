import React from 'react';

export default function RuleTable({ rules = [], onToggle, onDelete }) {
  return (
    <div>
      <h3 style={{ marginTop: 0 }}>Active Rules</h3>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Category</th>
              <th>Failure</th>
              <th>Value</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rules.map(rule => (
              <tr key={rule.id} className={!rule.enabled ? 'muted' : ''}>
                <td className="mono" title={rule.target}>{rule.target}</td>
                <td><span className={`chip ${rule.category}`}>{rule.category || '—'}</span></td>
                <td>
                  {rule.failure?.type && (
                    <span className={`chip ${rule.failure.type}`}>{rule.failure.type.toUpperCase()}</span>
                  )}
                </td>
                <td>
                  {rule.failure?.type === 'latency' && `${rule.failure.latencyMs}ms`}
                  {rule.failure?.type === 'error' && rule.failure.errorCode}
                </td>
                <td>
                  <button className={`switch ${rule.enabled ? 'on' : 'off'}`} onClick={() => onToggle(rule)}>
                    <span className="knob" />
                  </button>
                </td>
                <td>
                  <button className="btn danger" onClick={() => onDelete(rule.id)}>Delete</button>
                </td>
              </tr>
            ))}
            {rules.length === 0 && (
              <tr>
                <td colSpan="6" className="empty">No rules yet. Create one from the forms above.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
