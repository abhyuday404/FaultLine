import { useState, useEffect } from 'react';

// Styles are embedded directly into the component to make it self-contained.

const styles = `
  @import url('https://fonts.googleapis.com/css2?family=Poppins:wght@500;700&display=swap');
  body {
    font-family: 'Poppins', sans-serif;
    font-weight: 500;
    background-color: #262626;
    color: #e0e0e0;
    width: 100%;
    justify-content: center;
    margin: 0;
    padding: 0;
  }
  .container {
    max-width: 1000px;
    margin: 0 auto;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    justify-content: center;

  }
  header {
    text-align: center;
    margin-bottom: 3rem;
  }
  header h1 {
    color: #ffffff;
    font-size: 2.5rem;
    font-weight: 900;
  }
  header p {
    font-size: 1.1rem;
    color: #ffffff;
  }
  .cards-container {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
    gap: 2rem;
    justify-content: center;
  }
  .card {
    background-color: #1e1e1e;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    text-align: center;    
  }
  .api-card h3 {
    margin-top: 0;
    color: #ffffff;
    font-weight: 400;
  }
  .api-card p {
    color: #ffffff;
    flex-grow: 1;
  }
  .card button {
    padding: 0.8rem 1.5rem;
    font-size: 1rem;
    background-color: #5C5555;
    color: #ffffff;
    border: none;
    border-radius: 5px;
    cursor: pointer;
    transition: background-color 0.2s;
    margin-bottom: 1rem;
  }
  .card button:hover {
    background-color: #018786;
  }
  .response-box {
    background-color: #2c2c2c;
    border-radius: 5px;
    padding: 1rem;
    min-height: 150px;
    font-family: 'Poppins', sans-serif;
    font-size: 0.85rem;
    white-space: pre-wrap;
    word-break: break-all;
    text-align: center;
  }
  .response-box .error { color: #cf6679; }
  .response-box .success { color: #90ee90; }
`;

const Card = ({ title, description, endpoint, type = 'api' }) => {
  const [status, setStatus] = useState({ state: 'idle', data: null, time: 0 });

  const toTitleCase = (s) => s
    .replace(/[\-_]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\b\w/g, (c) => c.toUpperCase());

  const deriveTitleFromEndpoint = (ep) => {
    try {
      const u = new URL(ep);
      const parts = u.pathname.split('/').filter(Boolean);
      const last = parts[parts.length - 1] || '/';
      const friendly = `${u.hostname} • ${toTitleCase(last)}`;
      return friendly;
    } catch (_) {
      // If ep isn't a full URL, fall back to last path chunk
      const chunks = (ep || '').split('/').filter(Boolean);
      const last = chunks[chunks.length - 1] || ep || 'Request';
      return toTitleCase(last);
    }
  };

  const fetchData = async () => {
    const startTime = Date.now();
    setStatus({ state: 'loading', data: null, time: 0 });

    try {
      // The URL is constructed to go through the proxy to a live internet API
      const controller = new AbortController();
      const timeoutMs = type === 'database' ? 8000 : 15000; // shorter timeout for DB queries
      const timeoutId = setTimeout(() => controller.abort('Client-side timeout'), timeoutMs);
      const response = await fetch(`http://localhost:8080/${endpoint}`, { signal: controller.signal });
      clearTimeout(timeoutId);
      console.log("Trying to fetch from:- ");
      console.log(`http://localhost:8080/${endpoint}`);
      const timeTaken = Date.now() - startTime;
      
      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }
      
      const data = await response.json();
      setStatus({ state: 'success', data, time: timeTaken });

    } catch (error) {
      const timeTaken = Date.now() - startTime;
      const msg = error?.name === 'AbortError' ? 'Request aborted (timeout)' : (error?.message || String(error));
      setStatus({ state: 'error', data: msg, time: timeTaken });
    }
  };

  const cardClassName = `card ${type === 'database' ? 'database' : ''}`;

  const effectiveTitle = type === 'database' ? deriveTitleFromEndpoint(endpoint) : (title || deriveTitleFromEndpoint(endpoint));

  return (
    <div className={cardClassName}>
      <h3>{effectiveTitle}</h3>
      <p>{description}</p>
      <button onClick={fetchData}>
        {type === 'database' ? 'Query DB' : 'Fetch'} /{endpoint.split('/').pop()}
      </button>
      <div className="response-box">
        {status.state === 'idle' && <p>Click the button to {type === 'database' ? 'execute database query' : 'fetch data'}...</p>}
        {status.state === 'loading' && <p>{type === 'database' ? 'Executing query...' : 'Loading...'}</p>}
        {status.state === 'error' && (
          <div className="error">
            <p>{type === 'database' ? 'DATABASE ERROR!' : 'ERROR!'} (Took {status.time}ms)</p>
            <pre>{type === 'database' ? `DB Connection Failed: ${status.data}` : status.data}</pre>
          </div>
        )}
        {status.state === 'success' && (
          <div className="success">
            <p>{type === 'database' ? 'QUERY SUCCESS!' : 'SUCCESS!'} (Took {status.time}ms)</p>
            <pre>{JSON.stringify(status.data, null, 2).substring(0, 300)}...</pre>
          </div>
        )}
      </div>
    </div>
  );
};

function App() {
  const [dbTargets, setDbTargets] = useState([]);

  const fetchDbTargets = async () => {
    try {
      const res = await fetch('http://localhost:8081/api/rules');
      if (!res.ok) return;
      const data = await res.json();
      const all = Array.isArray(data) ? data : [];
      const dbRules = all.filter(r => r?.category === 'database');
      const uniqueTargets = Array.from(new Set(
        dbRules
          .map(r => r?.target)
          .filter(t => typeof t === 'string' && /^https?:\/\//.test(t))
      ));
      setDbTargets(uniqueTargets);
    } catch (_) {
      // non-blocking; fallback to defaults
    }
  };

  useEffect(() => {
    // Load rules from Control API and derive database targets dynamically
    fetchDbTargets();
  }, []);

  return (
    <>
      <style>{styles}</style>
      <div className="container">
        <header>
          <h1>FaultLine Showcase</h1>
          <p>This app makes API calls and simulates database operations through the FaultLine proxy. Use the Control Panel to inject failures and see the effects on both web APIs and database queries.</p>
        </header>
        <div style={{ margin: '0 0 1rem 0', display: 'flex', justifyContent: 'flex-end' }}>
          <button onClick={fetchDbTargets} style={{ padding: '0.5rem 0.75rem', cursor: 'pointer' }}>
            Refresh DB Endpoints
          </button>
        </div>
        <main className="cards-container">
          <Card 
            title="Fetch Users"
            description="Fetches a list of users from the live JSONPlaceholder API."
            endpoint="https://jsonplaceholder.typicode.com/users"
          />
          <Card 
            title="Fetch Posts"
            description="Fetches a list of posts. A good target for error injection."
            endpoint="https://jsonplaceholder.typicode.com/posts"
          />
          {dbTargets.length > 0 ? (
            dbTargets.map((t) => (
              <Card
                key={t}
                description="Database query via HTTP endpoint. Adjust failures in Control Panel."
                endpoint={t}
                type="database"
              />
            ))
          ) : (
            <>
              <Card 
                description="Simulates fetching user records from a database. Perfect for testing database timeout and connection errors."
                endpoint="https://jsonplaceholder.typicode.com/users?_limit=10"
                type="database"
              />
              <Card 
                description="Represents a complex database query that might fail due to connection issues or slow queries."
                endpoint="https://jsonplaceholder.typicode.com/comments?_limit=5"
                type="database"
              />
              <Card 
                description="Simulates database album/media lookups that could experience network partitions or database unavailability."
                endpoint="https://jsonplaceholder.typicode.com/albums"
                type="database"
              />
            </>
          )}
        </main>
      </div>
    </>
  );
}

export default App;

