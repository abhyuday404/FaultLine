import { useState } from 'react';

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
  .api-card {
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
  .api-card button {
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
  .api-card button:hover {
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

