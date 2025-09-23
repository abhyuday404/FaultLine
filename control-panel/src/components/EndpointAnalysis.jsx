import React, { useState } from 'react';

export default function EndpointAnalysis({ analyze, loading, error, endpoints, onCreateRule }) {
  const [copiedRow, setCopiedRow] = useState(null);
  const COPIED_MS = 1400;

  const copyToClipboard = async (text) => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      } else {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.setAttribute('readonly', '');
        ta.style.position = 'absolute';
        ta.style.left = '-9999px';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
    } catch (e) {
      console.error('Failed to copy to clipboard', e);
    }
  };

  const copyAllUrls = () => {
    const urls = (endpoints || [])
      .map(e => e?.url)
      .filter(Boolean);
    const unique = Array.from(new Set(urls));
    copyToClipboard(unique.join('\n'));
  };

  return (
    <div className="card">
      <div className="toolbar">
        <button className="btn primary" onClick={analyze} disabled={loading}>
          {loading ? 'Analyzing…' : 'Analyze Showcase App'}
        </button>
      </div>

      {error && (
        <div className="alert error">{error}</div>
      )}

      {endpoints?.length > 0 ? (
        <>
          <div className="table-wrap">
            <table className="table small">
              <thead>
                <tr>
                  <th>Method</th>
                  <th>URL</th>
                  <th>File</th>
                  <th>Line</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((endpoint, idx) => (
                  <tr key={idx}>
                    <td><span className={`chip method ${endpoint.method?.toLowerCase()}`}>{endpoint.method}</span></td>
                    <td className="mono ellipsis" title={`Click to copy → ${endpoint.method} ${endpoint.url}`}>
                      <span
                        role="button"
                        tabIndex={0}
                        onClick={async () => {
                          await copyToClipboard(endpoint.url);
                          setCopiedRow(idx);
                          window.clearTimeout(window.__flCopiedTimer);
                          window.__flCopiedTimer = window.setTimeout(() => setCopiedRow(null), COPIED_MS);
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            copyToClipboard(endpoint.url).then(() => {
                              setCopiedRow(idx);
                              window.clearTimeout(window.__flCopiedTimer);
                              window.__flCopiedTimer = window.setTimeout(() => setCopiedRow(null), COPIED_MS);
                            });
                          }
                        }}
                        style={{ cursor: 'pointer' }}
                      >
                        {endpoint.url}
                      </span>
                      {copiedRow === idx && (
                        <span className="chip success" style={{ marginLeft: '0.5rem' }}>Copied!</span>
                      )}
                    </td>
                    <td className="muted">
                      {endpoint.file ? endpoint.file.replace(/^.*[\\\\\/]/, '') : 'N/A'}
                    </td>
                    <td className="muted">{endpoint.line || 'N/A'}</td>
                    <td>
                      <button className="btn success" onClick={() => onCreateRule(endpoint)}>Create Rule</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="toolbar" style={{ marginTop: '0.5rem' }}>
            <button className="btn" onClick={copyAllUrls}>Copy all URLs</button>
          </div>
          <div className="alert info">
            <strong>Analysis Summary:</strong> Found {endpoints.length} endpoint{endpoints.length !== 1 ? 's' : ''} in use. Use these to create targeted failure injection rules.
          </div>
        </>
      ) : (
        !loading && !error && (
          <div className="empty center">Click Analyze to scan your source code for API endpoints.</div>
        )
      )}
    </div>
  );
}
