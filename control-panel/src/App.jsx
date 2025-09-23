import React, { useState, useEffect, useCallback } from 'react';
import './App.css';
import Sidebar from './components/Sidebar.jsx';
import Header from './components/Header.jsx';
import StatCard from './components/StatCard.jsx';
import RuleTable from './components/RuleTable.jsx';
import EndpointAnalysis from './components/EndpointAnalysis.jsx';

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
  const [activeSection, setActiveSection] = useState('dashboard');
  
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

  // Derived metrics
  const totalRules = rules.length;
  const enabledRules = rules.filter(r => r.enabled).length;
  const apiRules = rules.filter(r => r.category === 'api').length;
  const dbRules = rules.filter(r => r.category === 'database').length;

  if (loading) return <div className="center-page">Loading...</div>;
  if (error) return <div className="center-page error">Error: {error}</div>;

  return (
    <div className="layout">
      <Sidebar active={activeSection} onNavigate={setActiveSection} />
      <div className="main">
        <Header title="FaultLine Control Panel" subtitle="Inject failures. Test resilience. Gain confidence." />

        {activeSection === 'dashboard' && (
          <section className="section">
            <h2 className="section-title">Overview</h2>
            <div className="stats-grid">
              <StatCard label="Total Rules" value={totalRules} />
              <StatCard label="Enabled Rules" value={enabledRules} trend="up" />
              <StatCard label="API Rules" value={apiRules} />
              <StatCard label="DB Rules" value={dbRules} />
            </div>
          </section>
        )}

        {activeSection === 'rules' && (
          <section className="section">
            <h2 className="section-title">Manage Rules</h2>
            <div className="grid-2">
              <div className="card">
                <h3>Add API Failure Rule</h3>
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
                  <button className="btn primary" type="submit">Add Rule</button>
                </form>
              </div>

              <div className="card warn">
                <h3>Database Failure Rule</h3>
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
                    <label htmlFor="dbType">Failure Type</label>
                    <select id="dbType" name="type" value={newDbRule.type} onChange={handleDbInputChange}>
                      <option value="timeout">Timeout</option>
                      <option value="error">Error</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label htmlFor="dbValue">{newDbRule.type === 'timeout' ? 'Timeout (ms)' : 'HTTP Status Code'}</label>
                    <input type="number" id="dbValue" name="value" value={newDbRule.value} onChange={handleDbInputChange} placeholder={newDbRule.type === 'timeout' ? '5000' : '503'} />
                  </div>
                  <button className="btn primary" type="submit">Add DB Rule</button>
                </form>
              </div>
            </div>

            <div className="card">
              <RuleTable rules={rules} onToggle={handleUpdateRule} onDelete={handleDeleteRule} />
            </div>
          </section>
        )}

        {activeSection === 'analysis' && (
          <section className="section">
            <h2 className="section-title">Source Code Analysis</h2>
            <EndpointAnalysis
              analyze={() => analyzeCodeEndpoints('./showcase-app')}
              loading={codeAnalysisLoading}
              error={codeAnalysisError}
              endpoints={codeEndpoints}
              onCreateRule={createRuleFromCodeEndpoint}
            />
          </section>
        )}
      </div>
    </div>
  );
}

export default App;