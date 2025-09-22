import { useState, useEffect } from 'react'
import './App.css'

// Configuration
const USE_FAULTLINE_PROXY = true; // Set to true to route requests through FaultLine proxy
const FAULTLINE_PROXY_URL = 'http://localhost:8080';

// API endpoints for different services
const API_ENDPOINTS = {
  users: 'https://jsonplaceholder.typicode.com/users',
  posts: 'https://jsonplaceholder.typicode.com/posts',
  quotes: 'https://zenquotes.io/api/random',
  facts: 'https://uselessfacts.jsph.pl/random.json?language=en',
  faultlineRules: 'http://localhost:8081/api/rules',
  catImage: 'https://api.thecatapi.com/v1/images/search'
}

// Helper function to get the correct URL (direct or via FaultLine proxy)
const getProxyUrl = (originalUrl) => {
  if (USE_FAULTLINE_PROXY && !originalUrl.includes('localhost:8081')) {
    return `${FAULTLINE_PROXY_URL}/${originalUrl}`;
  }
  return originalUrl;
}

function App() {
  // State for different data sections
  const [users, setUsers] = useState([])
  const [posts, setPosts] = useState([])
  const [quote, setQuote] = useState(null)
  const [fact, setFact] = useState(null)
  const [faultlineStatus, setFaultlineStatus] = useState(null)
  const [catImage, setCatImage] = useState(null)
  
  // Database simulation state
  const [analytics, setAnalytics] = useState(null)
  const [userProfiles, setUserProfiles] = useState([])
  const [orders, setOrders] = useState([])
  
  // Loading states
  const [loading, setLoading] = useState({})
  const [errors, setErrors] = useState({})
  const [latency, setLatency] = useState({})

  // Generic fetch function with error handling and latency measurement
  const fetchDataWithLatency = async (url, setter, key, options = {}) => {
    setLoading(prev => ({ ...prev, [key]: true }))
    setErrors(prev => ({ ...prev, [key]: null }))
    setLatency(prev => ({ ...prev, [key]: null }))
    
    const startTime = performance.now()
    const finalUrl = getProxyUrl(url)
    
    console.log(`[${key}] Fetching from: ${finalUrl}`)
    
    try {
      const response = await fetch(finalUrl, {
        timeout: 10000,
        ...options
      })
      
      const endTime = performance.now()
      const responseTime = Math.round(endTime - startTime)
      setLatency(prev => ({ ...prev, [key]: responseTime }))
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      
      const data = await response.json()
      setter(data)
    } catch (error) {
      const endTime = performance.now()
      const responseTime = Math.round(endTime - startTime)
      setLatency(prev => ({ ...prev, [key]: responseTime }))
      
      console.error(`Error fetching ${key}:`, error)
      
      let errorMessage = error.message
      if (error.message.includes('fetch')) {
        errorMessage = 'Network error - check if FaultLine is intercepting this request'
      } else if (error.message.includes('HTTP')) {
        errorMessage = `${error.message} - Server returned an error`
      }
      
      setErrors(prev => ({ 
        ...prev, 
        [key]: errorMessage
      }))
    } finally {
      setLoading(prev => ({ ...prev, [key]: false }))
    }
  }

  // Simulate database calls (in a real app, these would be actual DB queries)
  const simulateDbCall = async (type) => {
    setLoading(prev => ({ ...prev, [type]: true }))
    setErrors(prev => ({ ...prev, [type]: null }))
    
    try {
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 800 + Math.random() * 1200))
      
      // Generate mock data based on type
      switch (type) {
        case 'analytics':
          setAnalytics({
            pageViews: Math.floor(Math.random() * 10000) + 5000,
            uniqueVisitors: Math.floor(Math.random() * 3000) + 1500,
            bounceRate: (Math.random() * 30 + 20).toFixed(1),
            avgSessionTime: `${Math.floor(Math.random() * 5) + 2}:${Math.floor(Math.random() * 60).toString().padStart(2, '0')}`
          })
          break
        case 'userProfiles':
          setUserProfiles([
            { id: 1, name: 'Alice Johnson', email: 'alice@example.com', status: 'active', lastLogin: '2025-09-22' },
            { id: 2, name: 'Bob Smith', email: 'bob@example.com', status: 'inactive', lastLogin: '2025-09-20' },
            { id: 3, name: 'Carol Brown', email: 'carol@example.com', status: 'active', lastLogin: '2025-09-23' }
          ])
          break
        case 'orders':
          setOrders([
            { id: 'ORD-001', customer: 'John Doe', amount: '$245.99', status: 'completed', date: '2025-09-22' },
            { id: 'ORD-002', customer: 'Jane Smith', amount: '$89.50', status: 'processing', date: '2025-09-23' },
            { id: 'ORD-003', customer: 'Mike Johnson', amount: '$156.75', status: 'shipped', date: '2025-09-21' }
          ])
          break
      }
    } catch (error) {
      setErrors(prev => ({ ...prev, [type]: 'Database connection failed' }))
    } finally {
      setLoading(prev => ({ ...prev, [type]: false }))
    }
  }

  // Load initial data
  useEffect(() => {
    // Direct fetch calls for code analyzer detection (these will be overridden by proper calls below)
    // fetch('https://jsonplaceholder.typicode.com/users')
    // fetch('https://jsonplaceholder.typicode.com/posts') 
    // fetch('https://zenquotes.io/api/random')
    // fetch('https://uselessfacts.jsph.pl/random.json?language=en')
    // fetch('http://localhost:8081/api/rules')
    // fetch('https://api.thecatapi.com/v1/images/search')
    
    // Load real API data with proper error handling and latency measurement
    fetchDataWithLatency('https://jsonplaceholder.typicode.com/users', setUsers, 'users')
    fetchDataWithLatency('https://jsonplaceholder.typicode.com/posts', 
      (data) => setPosts(data.slice(0, 5)), 'posts')
    fetchDataWithLatency('https://zenquotes.io/api/random', 
      (data) => setQuote(data[0]), 'quote')
    fetchDataWithLatency('https://uselessfacts.jsph.pl/random.json?language=en', setFact, 'fact')
    fetchDataWithLatency('http://localhost:8081/api/rules', setFaultlineStatus, 'faultline')
    fetchDataWithLatency('https://api.thecatapi.com/v1/images/search', 
      (data) => setCatImage(data[0]), 'catImage')
    
    // Simulate database calls
    simulateDbCall('analytics')
    simulateDbCall('userProfiles')
    simulateDbCall('orders')
  }, [])

  // Refresh functions for manual testing with latency measurement
  const refreshData = async (type) => {
    switch (type) {
      case 'users':
        await fetchDataWithLatency('https://jsonplaceholder.typicode.com/users', setUsers, 'users')
        break
      case 'posts':
        await fetchDataWithLatency('https://jsonplaceholder.typicode.com/posts', 
          (data) => setPosts(data.slice(0, 5)), 'posts')
        break
      case 'quote':
        await fetchDataWithLatency('https://zenquotes.io/api/random', 
          (data) => setQuote(data[0]), 'quote')
        break
      case 'fact':
        await fetchDataWithLatency('https://uselessfacts.jsph.pl/random.json?language=en', setFact, 'fact')
        break
      case 'faultline':
        await fetchDataWithLatency('http://localhost:8081/api/rules', setFaultlineStatus, 'faultline')
        break
      case 'catImage':
        await fetchDataWithLatency('https://api.thecatapi.com/v1/images/search', 
          (data) => setCatImage(data[0]), 'catImage')
        break
      default:
        simulateDbCall(type)
    }
  }

  // Direct fetch examples for code analyzer (not actually called)
  const demoFetchCalls = () => {
    fetch('https://jsonplaceholder.typicode.com/users')
    fetch('https://jsonplaceholder.typicode.com/posts') 
    fetch('https://zenquotes.io/api/random')
    fetch('https://uselessfacts.jsph.pl/random.json?language=en')
    fetch('http://localhost:8081/api/rules')
    fetch('https://api.thecatapi.com/v1/images/search')
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>🚀 FaultLine Showcase</h1>
        <p>A minimalistic demo app for testing failure injection scenarios</p>
        <div className="proxy-status">
          {USE_FAULTLINE_PROXY ? (
            <span className="status-indicator success">🔗 Using FaultLine Proxy (Port 8080)</span>
          ) : (
            <span className="status-indicator error">🔗 Direct Requests (No Proxy)</span>
          )}
        </div>
        <div className="faultline-status">
          {loading.faultline ? (
            <span className="status-indicator loading">🔄 Checking FaultLine...</span>
          ) : errors.faultline ? (
            <span className="status-indicator error">❌ FaultLine Offline</span>
          ) : (
            <span className="status-indicator success">✅ FaultLine Connected</span>
          )}
        </div>
      </header>

      <main className="app-main">
        {/* API Data Section */}
        <section className="data-section">
          <h2>📡 Live API Data</h2>
          <div className="cards-grid">
            
            {/* Users Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>👥 Users</h3>
                <button onClick={() => refreshData('users')} disabled={loading.users}>
                  {loading.users ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.users ? (
                <div className="error-message">{errors.users}</div>
              ) : (
                <div className="card-content">
                  <div className="metric">
                    {users.length} users loaded
                    {latency.users && (
                      <span className="latency"> • {latency.users}ms</span>
                    )}
                  </div>
                  <div className="data-preview">
                    {users.slice(0, 3).map(user => (
                      <div key={user.id} className="data-item">
                        <strong>{user.name}</strong> - {user.email}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Posts Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>📝 Recent Posts</h3>
                <button onClick={() => refreshData('posts')} disabled={loading.posts}>
                  {loading.posts ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.posts ? (
                <div className="error-message">{errors.posts}</div>
              ) : (
                <div className="card-content">
                  <div className="metric">
                    {posts.length} posts loaded
                    {latency.posts && (
                      <span className="latency"> • {latency.posts}ms</span>
                    )}
                  </div>
                  <div className="data-preview">
                    {posts.slice(0, 2).map(post => (
                      <div key={post.id} className="data-item">
                        <strong>{post.title.substring(0, 30)}...</strong>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Quote Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>💭 Daily Quote</h3>
                <button onClick={() => refreshData('quote')} disabled={loading.quote}>
                  {loading.quote ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.quote ? (
                <div className="error-message">{errors.quote}</div>
              ) : quote ? (
                <div className="card-content">
                  <div className="quote-text">"{quote.q}"</div>
                  <div className="quote-author">— {quote.a}</div>
                  {latency.quote && (
                    <div className="latency-info">Loaded in {latency.quote}ms</div>
                  )}
                </div>
              ) : (
                <div className="loading-placeholder">Loading quote...</div>
              )}
            </div>

            {/* Fun Fact Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>🎯 Random Fact</h3>
                <button onClick={() => refreshData('fact')} disabled={loading.fact}>
                  {loading.fact ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.fact ? (
                <div className="error-message">{errors.fact}</div>
              ) : fact ? (
                <div className="card-content">
                  <div className="fact-text">{fact.text}</div>
                  {latency.fact && (
                    <div className="latency-info">Loaded in {latency.fact}ms</div>
                  )}
                </div>
              ) : (
                <div className="loading-placeholder">Loading fact...</div>
              )}
            </div>

            {/* Cat Image Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>🐱 Random Cat</h3>
                <button onClick={() => refreshData('catImage')} disabled={loading.catImage}>
                  {loading.catImage ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.catImage ? (
                <div className="error-message">{errors.catImage}</div>
              ) : catImage ? (
                <div className="card-content">
                  <div className="cat-image-container">
                    <img 
                      src={catImage.url} 
                      alt="Random cat" 
                      className="cat-image"
                      onError={(e) => {
                        e.target.style.display = 'none';
                        setErrors(prev => ({ ...prev, catImage: 'Failed to load image' }));
                      }}
                    />
                  </div>
                  {latency.catImage && (
                    <div className="latency-info">Loaded in {latency.catImage}ms</div>
                  )}
                </div>
              ) : (
                <div className="loading-placeholder">Loading cat image...</div>
              )}
            </div>

          </div>
        </section>

        {/* Database Section */}
        <section className="data-section">
          <h2>🗄️ Database Queries</h2>
          <div className="cards-grid">
            
            {/* Analytics Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>📊 Analytics</h3>
                <button onClick={() => refreshData('analytics')} disabled={loading.analytics}>
                  {loading.analytics ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.analytics ? (
                <div className="error-message">{errors.analytics}</div>
              ) : analytics ? (
                <div className="card-content">
                  <div className="metrics-grid">
                    <div className="metric-item">
                      <div className="metric-value">{analytics.pageViews.toLocaleString()}</div>
                      <div className="metric-label">Page Views</div>
                    </div>
                    <div className="metric-item">
                      <div className="metric-value">{analytics.uniqueVisitors.toLocaleString()}</div>
                      <div className="metric-label">Unique Visitors</div>
                    </div>
                    <div className="metric-item">
                      <div className="metric-value">{analytics.bounceRate}%</div>
                      <div className="metric-label">Bounce Rate</div>
                    </div>
                    <div className="metric-item">
                      <div className="metric-value">{analytics.avgSessionTime}</div>
                      <div className="metric-label">Avg Session</div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="loading-placeholder">Loading analytics...</div>
              )}
            </div>

            {/* User Profiles Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>👤 User Profiles</h3>
                <button onClick={() => refreshData('userProfiles')} disabled={loading.userProfiles}>
                  {loading.userProfiles ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.userProfiles ? (
                <div className="error-message">{errors.userProfiles}</div>
              ) : (
                <div className="card-content">
                  <div className="table-container">
                    {userProfiles.map(profile => (
                      <div key={profile.id} className="profile-item">
                        <div className="profile-info">
                          <strong>{profile.name}</strong>
                          <span className={`status ${profile.status}`}>{profile.status}</span>
                        </div>
                        <div className="profile-meta">Last login: {profile.lastLogin}</div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Orders Card */}
            <div className="data-card">
              <div className="card-header">
                <h3>🛒 Recent Orders</h3>
                <button onClick={() => refreshData('orders')} disabled={loading.orders}>
                  {loading.orders ? '🔄' : '🔃'}
                </button>
              </div>
              {errors.orders ? (
                <div className="error-message">{errors.orders}</div>
              ) : (
                <div className="card-content">
                  <div className="table-container">
                    {orders.map(order => (
                      <div key={order.id} className="order-item">
                        <div className="order-info">
                          <strong>{order.id}</strong>
                          <span className="amount">{order.amount}</span>
                        </div>
                        <div className="order-meta">
                          <span className={`status ${order.status}`}>{order.status}</span>
                          <span className="date">{order.date}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </section>

      </main>
    </div>
  )
}

export default App
