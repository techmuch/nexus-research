import { useEffect, useState } from 'react';
import { 
  ShellLayout, 
  initializeShell, 
  commandRegistry, 
  menuRegistry, 
  componentRegistry,
  useLayoutStore 
} from 'nexus-shell';

interface StatusResponse {
  status: string;
  uptime: string;
  version: string;
  db_connected: boolean;
}

interface AuthCheckResponse {
  authenticated: boolean;
  username?: string;
}

function DashboardTab() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const base = import.meta.env.BASE_URL;
      const res = await fetch(`${base}api/status`);
      if (res.status === 401) {
        window.location.reload(); // Force reload to trigger login redirect
        return;
      }
      const data = await res.json();
      setStatus(data);
    } catch (err) {
      console.error('Failed to fetch API status:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  return (
    <div className="p-6 bg-card text-card-foreground h-full overflow-y-auto font-sans flex flex-col gap-6">
      <div className="border-b border-border pb-4 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-primary">NEXUS Research Station</h2>
          <p className="text-sm text-muted-foreground mt-1">Autonomous Multi-Agent Orchestration & Analysis Workbench</p>
        </div>
        <button 
          onClick={fetchStatus}
          disabled={loading}
          className="px-4 py-2 bg-primary text-primary-foreground font-semibold rounded hover:bg-primary/95 transition-all text-sm shadow-sm"
        >
          {loading ? 'Refreshing...' : 'Refresh Status'}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">System State</span>
          <span className="text-2xl font-extrabold text-foreground mt-2">ACTIVE</span>
          <span className="text-xs text-muted-foreground mt-4">All core engines online</span>
        </div>

        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">Go API Connection</span>
          <span className="text-2xl font-extrabold text-green-500 mt-2">
            {status ? 'CONNECTED' : 'DISCONNECTED'}
          </span>
          <span className="text-xs text-muted-foreground mt-4">
            {status ? `Uptime: ${status.uptime}` : 'Checking server connectivity...'}
          </span>
        </div>

        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">Build Version</span>
          <span className="text-2xl font-extrabold text-foreground mt-2">{status?.version || '0.1.0'}</span>
          <span className="text-xs text-muted-foreground mt-4">NEXUS-shell Engine v0.1.19</span>
        </div>
      </div>

      <div className="bg-muted/30 border border-border/80 rounded-lg p-6 flex flex-col gap-4">
        <h3 className="text-lg font-semibold text-foreground border-b border-border/50 pb-2">Active Research Pipelines</h3>
        <ul className="divide-y divide-border/50">
          <li className="py-3 flex justify-between items-center">
            <div>
              <p className="font-semibold text-foreground text-sm">LLM Agent Evaluation Pipeline</p>
              <p className="text-xs text-muted-foreground">Contextual feedback loops, trajectory analysis</p>
            </div>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">Running</span>
          </li>
          <li className="py-3 flex justify-between items-center">
            <div>
              <p className="font-semibold text-foreground text-sm">Visual Argumentation Mapper</p>
              <p className="text-xs text-muted-foreground">Compendium-style dialogue map rendering</p>
            </div>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-emerald-400">Idle</span>
          </li>
        </ul>
      </div>
    </div>
  );
}

// Custom simple markdown parser component for documentation rendering
function SimpleMarkdown({ content }: { content: string }) {
  const lines = content.split('\n');
  const renderedElements: React.ReactNode[] = [];
  let inCodeBlock = false;
  let codeBlockContent: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (line.startsWith('```')) {
      if (inCodeBlock) {
        renderedElements.push(
          <pre key={`code-${i}`} className="bg-muted p-4 rounded-lg overflow-x-auto font-mono text-sm border border-border/80 text-foreground my-4">
            <code>{codeBlockContent.join('\n')}</code>
          </pre>
        );
        codeBlockContent = [];
        inCodeBlock = false;
      } else {
        inCodeBlock = true;
      }
      continue;
    }

    if (inCodeBlock) {
      codeBlockContent.push(line);
      continue;
    }

    if (line.startsWith('# ')) {
      renderedElements.push(
        <h1 key={`h1-${i}`} className="text-3xl font-extrabold text-primary border-b border-border/60 pb-3 mt-6 mb-4">
          {line.replace('# ', '')}
        </h1>
      );
    } else if (line.startsWith('## ')) {
      renderedElements.push(
        <h2 key={`h2-${i}`} className="text-xl font-bold text-foreground border-b border-border/30 pb-2 mt-5 mb-3">
          {line.replace('## ', '')}
        </h2>
      );
    } else if (line.startsWith('### ')) {
      renderedElements.push(
        <h3 key={`h3-${i}`} className="text-lg font-semibold text-foreground mt-4 mb-2">
          {line.replace('### ', '')}
        </h3>
      );
    } else if (line.startsWith('- ')) {
      renderedElements.push(
        <li key={`li-${i}`} className="ml-5 list-disc text-sm text-foreground my-1">
          {line.replace('- ', '')}
        </li>
      );
    } else if (line.trim() === '') {
      renderedElements.push(<div key={`space-${i}`} className="h-2" />);
    } else {
      renderedElements.push(
        <p key={`p-${i}`} className="text-sm text-muted-foreground leading-relaxed my-2">
          {line}
        </p>
      );
    }
  }

  return <div className="prose prose-sm dark:prose-invert max-w-none">{renderedElements}</div>;
}

function DocsTab() {
  const [docId, setDocId] = useState<'getting-started' | 'architecture'>('getting-started');
  const [content, setContent] = useState<string>('Loading documentation...');

  useEffect(() => {
    const fetchDoc = async () => {
      try {
        const base = import.meta.env.BASE_URL;
        const res = await fetch(`${base}docs/${docId}.md`);
        if (!res.ok) {
          throw new Error(`Failed to load ${docId}`);
        }
        const text = await res.text();
        setContent(text);
      } catch (err) {
        setContent(`Error: Failed to load ${docId} documentation page.`);
      }
    };
    fetchDoc();
  }, [docId]);

  return (
    <div className="flex h-full font-sans bg-card text-card-foreground">
      {/* Documentation Sidebar */}
      <div className="w-64 border-r border-border bg-muted/20 flex flex-col p-4 gap-2">
        <h3 className="text-xs font-semibold tracking-wider text-muted-foreground uppercase mb-3 px-2">Documentation</h3>
        <button
          onClick={() => setDocId('getting-started')}
          className={`text-left px-3 py-2 rounded text-sm transition-all ${
            docId === 'getting-started' 
              ? 'bg-primary text-primary-foreground font-semibold shadow-sm' 
              : 'hover:bg-muted text-muted-foreground hover:text-foreground'
          }`}
        >
          Getting Started
        </button>
        <button
          onClick={() => setDocId('architecture')}
          className={`text-left px-3 py-2 rounded text-sm transition-all ${
            docId === 'architecture' 
              ? 'bg-primary text-primary-foreground font-semibold shadow-sm' 
              : 'hover:bg-muted text-muted-foreground hover:text-foreground'
          }`}
        >
          Technical Architecture
        </button>
      </div>

      {/* Documentation Content Area */}
      <div className="flex-1 p-8 overflow-y-auto">
        <SimpleMarkdown content={content} />
      </div>
    </div>
  );
}

function LoginScreen({ onLoginSuccess }: { onLoginSuccess: (username: string) => void }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setError('Username and password are required');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const base = import.meta.env.BASE_URL;
      const res = await fetch(`${base}api/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const data = await res.json();
        setError(data.error || 'Authentication failed');
        return;
      }

      const data = await res.json();
      onLoginSuccess(data.username);
    } catch (err) {
      setError('Server connection error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="h-screen w-screen flex items-center justify-center bg-[#0D1117] text-white font-sans overflow-hidden relative">
      {/* Decorative backdrop elements */}
      <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] rounded-full bg-emerald-500/10 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] rounded-full bg-blue-500/10 blur-[120px] pointer-events-none" />

      {/* Login Card */}
      <div className="w-[420px] p-8 bg-[#161B22]/90 border border-gray-800 rounded-xl shadow-[0_8px_32px_0_rgba(0,0,0,0.5)] backdrop-blur-md relative z-10 flex flex-col gap-6 transition-all duration-300 hover:border-emerald-500/30">
        <div className="text-center">
          <h1 className="text-2xl font-extrabold tracking-wider text-emerald-500">NEXUS RESEARCH STATION</h1>
          <p className="text-xs text-gray-400 mt-2">Enter credentials to establish control terminal connection</p>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded p-3 text-sm text-red-400 font-medium">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          <div className="flex flex-col gap-2">
            <label className="text-xs font-semibold tracking-wide text-gray-300 uppercase">Username</label>
            <input 
              type="text" 
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="e.g. admin"
              disabled={loading}
              className="px-4 py-3 bg-[#0D1117] border border-gray-800 rounded text-sm text-white focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all placeholder:text-gray-600"
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-semibold tracking-wide text-gray-300 uppercase">Password</label>
            <input 
              type="password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              disabled={loading}
              className="px-4 py-3 bg-[#0D1117] border border-gray-800 rounded text-sm text-white focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all placeholder:text-gray-600"
            />
          </div>

          <button 
            type="submit"
            disabled={loading}
            className="w-full mt-2 py-3 bg-emerald-500 text-[#0D1117] font-bold tracking-wider rounded hover:bg-emerald-400 disabled:bg-emerald-600/50 disabled:text-gray-500 transition-all active:scale-[0.98]"
          >
            {loading ? 'CONNECTING...' : 'ESTABLISH LINK'}
          </button>
        </form>

        <div className="border-t border-gray-800/60 pt-4 text-center">
          <p className="text-[10px] text-gray-500 leading-normal">
            Admins must register user accounts locally using the CLI utility:
            <code className="block mt-2 p-1.5 bg-[#0D1117] border border-gray-800/80 rounded font-mono text-gray-400 text-[9px] select-all">
              nexus-research user create
            </code>
          </p>
        </div>
      </div>
    </div>
  );
}

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [user, setUser] = useState<string>('');

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const base = import.meta.env.BASE_URL;
        const res = await fetch(`${base}api/auth/check`);
        const data: AuthCheckResponse = await res.json();
        if (data.authenticated) {
          setIsAuthenticated(true);
          setUser(data.username || 'user');
        }
      } catch (err) {
        console.error('Failed to verify session:', err);
      } finally {
        setLoading(false);
      }
    };
    checkAuth();
  }, []);

  const handleLoginSuccess = (username: string) => {
    setIsAuthenticated(true);
    setUser(username);
  };

  useEffect(() => {
    if (isAuthenticated) {
      // Initialize core registries
      initializeShell();

      // Register Custom Tab Components
      componentRegistry.register('nexus-dashboard', DashboardTab);
      componentRegistry.register('nexus-docs', DocsTab);

      // Register Custom commands
      commandRegistry.registerCommand({
        id: 'app.ping_api',
        label: 'Ping Backend API',
        execute: async () => {
          try {
            const base = import.meta.env.BASE_URL;
            const res = await fetch(`${base}api/status`);
            const data = await res.json();
            alert(`API Response:\n\nStatus: ${data.status}\nUptime: ${data.uptime}\nVersion: ${data.version}\nDatabase connected: ${data.db_connected}`);
          } catch (err) {
            alert(`API Error: Failed to connect to server.`);
          }
        },
      });

      commandRegistry.registerCommand({
        id: 'app.open_docs',
        label: 'Open Documentation',
        execute: () => {
          const layoutState = useLayoutStore.getState();
          layoutState.addTab('nexus-docs', 'Documentation');
        },
      });

      // Add to the File and Help menus
      menuRegistry.registerMenu('File', {
        id: 'file.ping_api',
        label: 'Ping API Status',
        commandId: 'app.ping_api',
      });

      menuRegistry.registerMenu('Help', {
        id: 'help.open_docs',
        label: 'Open Documentation',
        commandId: 'app.open_docs',
      });

      // Automatically open the dashboard & docs tab on startup
      const layoutState = useLayoutStore.getState();
      layoutState.addTab('nexus-dashboard', 'Dashboard');
      layoutState.addTab('nexus-docs', 'Documentation');
    }
  }, [isAuthenticated]);

  if (loading) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-[#0D1117] text-emerald-500 font-mono text-sm">
        INITIALIZING CORE SYSTEM...
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginScreen onLoginSuccess={handleLoginSuccess} />;
  }

  return (
    <div style={{ width: '100vw', height: '100vh' }}>
      <ShellLayout 
        title={<div className="font-bold text-lg text-primary">NEXUS RESEARCH STATION - TERMINAL {user.toUpperCase()}</div>} 
      />
    </div>
  );
}

export default App;
