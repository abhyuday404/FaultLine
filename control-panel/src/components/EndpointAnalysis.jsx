import React from 'react';

export default function EndpointAnalysis({ analyze, loading, error, endpoints, onCreateRule }) {
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
                    <td className="mono ellipsis" title={endpoint.url}>{endpoint.url}</td>
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
