import { useState } from 'react';

// Styles are embedded directly into the component to make it self-contained.
const styles = `
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
    background-color: #121212;
    color: #e0e0e0;
    margin: 0;
    padding: 2rem;
  }
  .container {
    max-width: 1000px;
    margin: 0 auto;
  }
  header {
    text-align: center;
    margin-bottom: 3rem;
  }
  header h1 {
    color: #bb86fc;
    font-size: 2.5rem;
  }
  header p {
    font-size: 1.1rem;
    color: #a0a0a0;
  }
  .cards-container {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
    gap: 2rem;
  }
  .api-card {
    background-color: #1e1e1e;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
  }
  .api-card h3 {
    margin-top: 0;
    color: #03dac6;
  }
  .api-card p {
    color: #a0a0a0;
    flex-grow: 1;
  }
  .api-card button {
    padding: 0.8rem 1.5rem;
    font-size: 1rem;
    background-color: #03dac6;
    color: #121212;
    border: none;
    border-radius: 5px;
    cursor: pointer;
    transition: background-color 0.2s;
    margin-bottom: 1rem;
  }
  .api-card button:hover {
    background-color: #018786;
  }
  .response-box {
    background-color: #2c2c2c;
    border-radius: 5px;
    padding: 1rem;
    min-height: 150px;
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.85rem;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .response-box .error { color: #cf6679; }
  .response-box .success { color: #90ee90; }
`;

const ApiCard = ({ title, description, endpoint }) => {
  const [status, setStatus] = useState({ state: 'idle', data: null, time: 0 });

  const fetchData = async () => {
    const startTime = Date.now();
    setStatus({ state: 'loading', data: null, time: 0 });

    try {
      // The URL is constructed to go through the proxy to a live internet API
      const response = await fetch(`http://localhost:8080/${endpoint}`);
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
      setStatus({ state: 'error', data: error.message, time: timeTaken });
    }
  };

  return (
    <div className="api-card">
      <h3>{title}</h3>
      <p>{description}</p>
      <button onClick={fetchData}>Fetch /{endpoint.split('/').pop()}</button>
      <div className="response-box">
        {status.state === 'idle' && <p>Click the button to fetch data...</p>}
        {status.state === 'loading' && <p>Loading...</p>}
        {status.state === 'error' && (
          <div className="error">
            <p>ERROR! (Took {status.time}ms)</p>
            <pre>{status.data}</pre>
          </div>
        )}
        {status.state === 'success' && (
          <div className="success">
            <p>SUCCESS! (Took {status.time}ms)</p>
            <pre>{JSON.stringify(status.data, null, 2).substring(0, 300)}...</pre>
          </div>
        )}
      </div>
    </div>
  );
};

function App() {
  return (
    <>
      <style>{styles}</style>
      <div className="container">
        <header>
          <h1>FaultLine Showcase</h1>
          <p>This app makes API calls to a live public API (JSONPlaceholder) through the FaultLine proxy. Use the Control Panel to inject failures and see the effects here.</p>
        </header>
        <main className="cards-container">
          <ApiCard 
            title="Fetch Users"
            description="Fetches a list of users from the live JSONPlaceholder API."
            endpoint="https://jsonplaceholder.typicode.com/users"
          />
          <ApiCard 
            title="Fetch Posts"
            description="Fetches a list of posts. A good target for error injection."
            endpoint="https://jsonplaceholder.typicode.com/posts"
          />
        </main>
      </div>
    </>
  );
}

export default App;

