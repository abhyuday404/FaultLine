import React, { useState, useEffect, useCallback } from 'react';

// --- STYLES (previously in App.css) ---
const styles = `
  :root {
    --bg-color: #1a1a1a;
    --card-color: #2a2a2a;
    --border-color: #444;
    --text-color: #f0f0f0;
    --text-muted: #888;
    --accent-color: #4a90e2;
    --accent-hover: #5aa1f2;
    --red: #e94f4f;
    --green: #52b788;
    --orange: #f7b801;
    --font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  }
  body {
    font-family: var(--font-family);
    background-color: var(--bg-color);
    color: var(--text-color);
    width: 100%;
    justify-content: center;
    margin: 0;
    padding: 2rem;
  }
  .container { max-width: 1000px; margin: 0 auto; }
  h1, h2 { color: var(--text-color); border-bottom: 1px solid var(--border-color); padding-bottom: 0.5rem; }
  h1 { font-size: 2.5rem; }
  h2 { font-size: 1.75rem; margin-top: 2rem; }
  .card { background-color: var(--card-color); border-radius: 8px; border: 1px solid var(--border-color); padding: 1.5rem; margin-bottom: 1.5rem; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid var(--border-color); }
  th { color: var(--text-muted); font-weight: 600; }
  tr.disabled { color: var(--text-muted); text-decoration: line-through; }
  tr.disabled .badge { background-color: #555; }
  .badge { padding: 0.25rem 0.6rem; border-radius: 12px; font-size: 0.8rem; font-weight: 700; color: #fff; }
  .badge.error { background-color: var(--red); }
  .badge.latency { background-color: var(--orange); }
  .badge.flaky { background-color: var(--accent-color); }
  .card.database { border-left: 4px solid #ff9800; background-color: #2a2520; }
  .card.database h2 { color: #ffb74d; }
  .form-grid { display: grid; grid-template-columns: 1fr auto auto; gap: 1rem; align-items: flex-end; }
  .form-group { display: flex; flex-direction: column; }
  label { margin-bottom: 0.5rem; font-size: 0.9rem; color: var(--text-muted); }
  input, select, button {
    padding: 0.75rem;
    border-radius: 6px;
    background-color: #333;
    border: 1px solid var(--border-color);
    color: var(--text-color);
    font-size: 1rem;
    font-family: var(--font-family);
  }
  button {
    background-color: var(--accent-color);
    border-color: var(--accent-color);
    cursor: pointer;
    font-weight: 600;
    transition: background-color 0.2s;
  }
  button:hover { background-color: var(--accent-hover); }
  .toggle-btn { background: none; border: none; cursor: pointer; font-size: 1.5rem; padding: 0.5rem; }
  .delete-btn { background: none; border: none; cursor: pointer; color: var(--red); font-size: 1rem; padding: 0.5rem; }
`;

const API_URL = 'http://localhost:8081';

function App() {
  const [rules, setRules] = useState([]);
  const [newRule, setNewRule] = useState({
    target: 'http://localhost:3000/products',
    type: 'error',
    value: '503',
  });
  const [newDbRule, setNewDbRule] = useState({
    target: '',
    type: 'timeout',
    value: '5000',
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  // Code analysis state
  const [codeEndpoints, setCodeEndpoints] = useState([]);
  const [codeAnalysisLoading, setCodeAnalysisLoading] = useState(false);
  const [codeAnalysisError, setCodeAnalysisError] = useState(null);

  const fetchRules = useCallback(async () => {
    try {
      const response = await fetch(`${API_URL}/api/rules`);
      if (!response.ok) throw new Error('Failed to fetch rules');
      const data = await response.json();
      const all = data || [];
      // Show all rules in a single table regardless of category
      setRules(all);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  // Analyze source code for actual endpoints
  const analyzeCodeEndpoints = async (directory = './showcase-app') => {
    setCodeAnalysisLoading(true);
    setCodeAnalysisError(null);
    try {
      const response = await fetch(`${API_URL}/api/endpoints/analyze-code?directory=${encodeURIComponent(directory)}`);
      if (!response.ok) throw new Error('Failed to analyze code endpoints');
      const data = await response.json();
      
      setCodeEndpoints(data.endpoints || []);
    } catch (err) {
      setCodeAnalysisError(err.message);
    } finally {
      setCodeAnalysisLoading(false);
    }
  };

  // Create rule from code endpoint
  const createRuleFromCodeEndpoint = async (endpoint) => {
    try {
      const newRulePayload = {
        target: endpoint.url,
        category: 'api',
        failure: {
          type: 'latency',
          latencyMs: 2000,
        },
      };

      const response = await fetch(`${API_URL}/api/rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newRulePayload),
      });

      if (!response.ok) throw new Error('Failed to create rule');
      
      fetchRules(); // Refresh rules list
      alert('Rule created successfully!');
    } catch (err) {
      alert('Failed to create rule: ' + err.message);
    }
  };

  const handleDbInputChange = (e) => {
    const { name, value } = e.target;
    setNewDbRule(prev => ({ ...prev, [name]: value }));
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setNewRule(prev => ({ ...prev, [name]: value }));
  };

  const handleAddDbRule = async (e) => {
    e.preventDefault();
    try {
      // Map UI DB types to backend-supported failure types
      const mappedType = newDbRule.type === 'error' ? 'error' : 'latency';
      const latencyMs = newDbRule.type === 'timeout' ? parseInt(newDbRule.value, 10) : 0;
      const errorCode = newDbRule.type === 'error' ? parseInt(newDbRule.value, 10) : 0;

      const newDbRulePayload = {
        target: newDbRule.target,
        category: 'database',
        failure: {
          type: mappedType,
          latencyMs,
          errorCode,
        },
      };

      const response = await fetch(`${API_URL}/api/rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newDbRulePayload),
      });

      if (!response.ok) throw new Error('Failed to add database rule');
      const addedRule = await response.json();
      // Append to unified rules list
      setRules(prev => [...prev, addedRule]);
    } catch (err) {
      setError(err.message);
    }
  };

  const handleAddRule = async (e) => {
    e.preventDefault();
    try {
      // Standardize on camelCase for the payload
      const newRulePayload = {
        target: newRule.target,
        category: 'api',
        failure: {
          type: newRule.type,
          latencyMs: newRule.type === 'latency' ? parseInt(newRule.value, 10) : 0,
          errorCode: newRule.type === 'error' ? parseInt(newRule.value, 10) : 0,
        },
      };

      const response = await fetch(`${API_URL}/api/rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newRulePayload),
      });

      if (!response.ok) throw new Error('Failed to add rule');
      const addedRule = await response.json();
      setRules([...rules, addedRule]);
    } catch (err) {
      setError(err.message);
    }
  };

  // (DB update/delete handlers removed; unified table uses generic handlers below)

  const handleUpdateRule = async (rule) => {
    try {
      // The rule object from state is already camelCase
      const updatedPayload = { ...rule, enabled: !rule.enabled };

      const response = await fetch(`${API_URL}/api/rules/${rule.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updatedPayload),
      });

      if (!response.ok) throw new Error('Failed to update rule');
      const updatedRule = await response.json();
      setRules(rules.map(r => (r.id === updatedRule.id ? updatedRule : r)));
    } catch (err) {
      setError(err.message);
    }
  };

  const handleDeleteRule = async (id) => {
    try {
      const response = await fetch(`${API_URL}/api/rules/${id}`, { method: 'DELETE' });
      if (!response.ok) throw new Error('Failed to delete rule');
      setRules(rules.filter(r => r.id !== id));
    } catch (err) {
      setError(err.message);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <>
      <style>{styles}</style>
      <div className="container">
        <h1>FaultLine Control Panel</h1>
        
        <div className="card">
          <h2>Add API Failure Rule</h2>
          <form onSubmit={handleAddRule} className="form-grid">
            <div className="form-group">
              <label htmlFor="target">Target URL Prefix</label>
              <input type="text" id="target" name="target" value={newRule.target} onChange={handleInputChange} required />
            </div>
            <div className="form-group">
              <label htmlFor="type">Failure Type</label>
              <select id="type" name="type" value={newRule.type} onChange={handleInputChange}>
                <option value="error">Error Code</option>
                <option value="latency">Latency</option>
              </select>
            </div>
            <div className="form-group">
              <label htmlFor="value">{newRule.type === 'latency' ? 'Latency (ms)' : 'HTTP Status Code'}</label>
              <input type="number" id="value" name="value" value={newRule.value} onChange={handleInputChange} required />
            </div>
            <button type="submit">Add Rule</button>
          </form>
        </div>

        <div className="card database">
          <h2>🗄️ Add Database Failure Rule</h2>
          <form onSubmit={handleAddDbRule} className="form-grid">
            <div className="form-group">
              <label htmlFor="dbTarget">Database URL (HTTP/S via proxy)</label>
              <input
                type="url"
                id="dbTarget"
                name="target"
                placeholder="e.g. https://jsonplaceholder.typicode.com/users"
                value={newDbRule.target}
                onChange={handleDbInputChange}
                required
              />
            </div>
            <div className="form-group">
              <label htmlFor="dbType">Database Failure Type</label>
              <select 
                id="dbType" 
                name="type" 
                value={newDbRule.type} 
                onChange={handleDbInputChange}
                className="form-select"
              >
                  <option value="timeout">Timeout</option>
                  <option value="error">Error</option>
              </select>
            </div>
            <div className="form-group">
              <label htmlFor="dbValue">
                  {newDbRule.type === 'timeout' 
                      ? 'Timeout (ms)'
                      : 'HTTP Status Code'
                  }
              </label>
              <input
                  type="number"
                  id="dbValue"
                  name="value"
                  value={newDbRule.value}
                  onChange={handleDbInputChange}
                  className="form-input"
                  placeholder={newDbRule.type === 'timeout' ? '5000' : '503'}
              />
            </div>
            <button type="submit">Add DB Rule</button>
          </form>
        </div>

        <div className="card">
          <h2>Source Code Analysis (For API Endpoints Only)</h2>
          <p style={{ color: 'var(--text-muted)', marginBottom: '1rem' }}>
            Analyze actual source code to discover API endpoints being used in your application.
          </p>
          
          <div style={{ marginBottom: '1rem' }}>
            <button 
              onClick={() => analyzeCodeEndpoints('./showcase-app')}
              disabled={codeAnalysisLoading}
            >
              {codeAnalysisLoading ? '🔄 Analyzing...' : '🔍 Analyze Showcase App'}
            </button>
          </div>

          {codeAnalysisError && (
            <div style={{ color: 'var(--red)', marginBottom: '1rem', padding: '0.5rem', background: 'rgba(255,0,0,0.1)', borderRadius: '4px' }}>
              ❌ {codeAnalysisError}
            </div>
          )}

          {codeEndpoints.length > 0 && (
            <>
              <h3 style={{ marginBottom: '1rem', color: 'var(--accent)' }}>
                📋 Code Endpoints Found ({codeEndpoints.length})
              </h3>
              <div style={{ maxHeight: '400px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: '4px' }}>
                <table style={{ fontSize: '0.9rem' }}>
                  <thead>
                    <tr style={{ background: 'var(--card-bg)', position: 'sticky', top: 0 }}>
                      <th>Method</th>
                      <th>URL</th>
                      <th>File</th>
                      <th>Line</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {codeEndpoints.map((endpoint, index) => (
                      <tr key={index}>
                        <td>
                          <span style={{ 
                            padding: '2px 6px', 
                            borderRadius: '3px', 
                            fontSize: '0.8rem', 
                            fontWeight: 'bold',
                            background: endpoint.method === 'GET' ? 'var(--green)' : endpoint.method === 'POST' ? 'var(--blue)' : 'var(--yellow)',
                            color: 'white'
                          }}>
                            {endpoint.method}
                          </span>
                        </td>
                        <td style={{ maxWidth: '300px', wordBreak: 'break-all' }}>{endpoint.url}</td>
                        <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                          {endpoint.file ? endpoint.file.replace(/^.*[\\\/]/, '') : 'N/A'}
                        </td>
                        <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                          {endpoint.line || 'N/A'}
                        </td>
                        <td>
                          <button 
                            onClick={() => createRuleFromCodeEndpoint(endpoint)}
                            style={{ 
                              background: 'var(--green)', 
                              color: 'white', 
                              border: 'none', 
                              padding: '4px 8px', 
                              borderRadius: '3px',
                              fontSize: '0.8rem'
                            }}
                          >
                             Create Rule
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              
              <div style={{ marginTop: '1rem', padding: '0.75rem', background: 'rgba(0,255,0,0.1)', borderRadius: '4px', fontSize: '0.9rem' }}>
                <strong>💡 Analysis Summary:</strong>
                <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem' }}>
                  <li>Found {codeEndpoints.length} actual API endpoint{codeEndpoints.length !== 1 ? 's' : ''} in use</li>
                  <li>These are the endpoints your application actually calls</li>
                  <li>Use this data to create targeted failure injection rules</li>
                </ul>
              </div>
            </>
          )}

          {codeEndpoints.length === 0 && !codeAnalysisLoading && !codeAnalysisError && (
            <div style={{ color: 'var(--text-muted)', fontStyle: 'italic', textAlign: 'center', padding: '2rem' }}>
              Click "Analyze" to scan your source code for API endpoints.
              <br />
              This will show you the actual endpoints your application uses.
            </div>
          )}
        </div>

        <div className="card">
          <h2>Active Rules</h2>
          <table>
            <thead>
              <tr>
                <th>Target</th>
                <th>Failure Type</th>
                <th>Value</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.map(rule => (
                <tr key={rule.id} className={!rule.enabled ? 'disabled' : ''}>
                  <td>{rule.target}</td>
                  <td>
                    {rule.failure && rule.failure.type && (
                      <span className={`badge ${rule.failure.type}`}>
                        {rule.failure.type.toUpperCase()}
                      </span>
                    )}
                  </td>
                  <td>
                    {rule.failure?.type === 'latency' && `${rule.failure.latencyMs}ms`}
                    {rule.failure?.type === 'error' && rule.failure.errorCode}
                  </td>
                  <td>
                    <button onClick={() => handleUpdateRule(rule)} className="toggle-btn" title={rule.enabled ? "Disable Rule" : "Enable Rule"}>
                      {rule.enabled ? '🟢' : '⚪️'}
                    </button>
                  </td>
                  <td>
                    <button onClick={() => handleDeleteRule(rule.id)} className="delete-btn">
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

export default App;