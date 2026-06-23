import { useEffect, useState, useRef } from 'react';
import { 
  NexusWorkspaceShell, 
  initializeShell, 
  commandRegistry, 
  menuRegistry, 
  componentRegistry,
  useLayoutStore,
  useUserProfileStore
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

function UserProfileTab() {
  const [profile, setProfile] = useState({ full_name: '', title: '', avatar_data: '' });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const userProfileStore = useUserProfileStore();

  useEffect(() => {
    const base = import.meta.env.BASE_URL;
    fetch(`${base}api/profile`)
      .then(r => r.json())
      .then(data => {
        setProfile({
           full_name: data.full_name || '',
           title: data.title || '',
           avatar_data: data.avatar_data || ''
        });
        setLoading(false);
        userProfileStore.updateProfile({ 
           name: data.full_name || 'Anonymous', 
           role: data.title || ''
        });
        if (data.avatar_data) {
           userProfileStore.setCustomAvatar(data.avatar_data);
        }
      });
  }, []);

  const handleSave = async () => {
    setSaving(true);
    const base = import.meta.env.BASE_URL;
    await fetch(`${base}api/profile`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(profile)
    });
    setSaving(false);
    userProfileStore.updateProfile({ name: profile.full_name || 'Anonymous', role: profile.title });
    if (profile.avatar_data) {
       userProfileStore.setCustomAvatar(profile.avatar_data);
    } else {
       userProfileStore.clearCustomAvatar();
    }
  };

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onloadend = () => {
        setProfile(p => ({ ...p, avatar_data: reader.result as string }));
      };
      reader.readAsDataURL(file);
    }
  };

  if (loading) return <div className="p-8 text-white">Loading profile...</div>;

  return (
    <div className="p-8 h-full overflow-y-auto bg-background text-foreground font-sans flex items-center justify-center">
      <div className="max-w-4xl w-full bg-card border border-border rounded-xl shadow-xl flex flex-col md:flex-row overflow-hidden">
        
        {/* Left Pane - Avatar Area */}
        <div className="md:w-1/3 bg-muted/30 border-b md:border-b-0 md:border-r border-border p-8 flex flex-col items-center justify-center">
          <div className="w-40 h-40 rounded-full overflow-hidden border-2 border-border bg-muted relative group cursor-pointer shadow-inner mb-4">
            {profile.avatar_data ? (
              <img src={profile.avatar_data} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-muted-foreground font-mono text-xs">AVATAR</div>
            )}
            <input 
              type="file" 
              accept="image/*" 
              onChange={handleAvatarChange}
              className="absolute inset-0 opacity-0 cursor-pointer z-10" 
            />
            <div className="absolute inset-0 bg-background/60 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
              <span className="text-sm font-bold text-foreground tracking-wider">UPLOAD</span>
            </div>
          </div>
          <h3 className="text-lg font-bold text-card-foreground text-center line-clamp-1">{profile.full_name || 'Anonymous'}</h3>
          <p className="text-xs text-muted-foreground text-center mt-1 line-clamp-1">{profile.title || 'User Profile'}</p>
        </div>

        {/* Right Pane - Form Area */}
        <div className="md:w-2/3 p-8 flex flex-col justify-center">
          <div className="mb-8">
            <h2 className="text-sm font-bold text-muted-foreground uppercase tracking-wider mb-1">Account Settings</h2>
            <h1 className="text-2xl font-bold text-card-foreground">Profile Information</h1>
          </div>

          <div className="flex flex-col gap-6">
            <div>
              <label className="text-xs font-bold text-muted-foreground mb-2 block">Full Name</label>
              <input 
                type="text" 
                value={profile.full_name} 
                onChange={e => setProfile({...profile, full_name: e.target.value})}
                className="w-full bg-input border border-border rounded-md px-4 py-2.5 text-sm text-foreground focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/50 transition-all shadow-sm"
              />
            </div>
            <div>
              <label className="text-xs font-bold text-muted-foreground mb-2 block">Title / Role</label>
              <input 
                type="text" 
                value={profile.title} 
                onChange={e => setProfile({...profile, title: e.target.value})}
                className="w-full bg-input border border-border rounded-md px-4 py-2.5 text-sm text-foreground focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/50 transition-all shadow-sm"
              />
            </div>
          </div>

          <div className="mt-10 flex">
            <button 
              onClick={handleSave}
              disabled={saving}
              className="bg-primary hover:bg-primary/90 text-primary-foreground font-semibold py-2.5 px-6 rounded-md transition-colors shadow-sm"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>

      </div>
    </div>
  );
}

function WelcomeTab() {
  return (
    <div className="p-8 h-full overflow-y-auto bg-background text-foreground font-sans flex flex-col gap-6">
      <div className="max-w-3xl mx-auto w-full">
        <h1 className="text-3xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-emerald-400 mb-2">
          Welcome to Nexus Research
        </h1>
        <p className="text-lg text-muted-foreground mb-8 border-b border-border pb-6">
          Autonomous Multi-Agent Orchestration & Dialogue Mapping
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-card border border-border rounded-lg p-6 hover:bg-accent hover:text-accent-foreground transition-colors">
            <h2 className="text-xl font-bold text-card-foreground mb-3">Enterprise Database</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Nexus Research uses a robust Go + SQLite backend to ensure all of your layouts, dialogue maps, and arguments are safely persisted across sessions. Say goodbye to volatile local storage.
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6 hover:bg-accent hover:text-accent-foreground transition-colors">
            <h2 className="text-xl font-bold text-card-foreground mb-3">Multi-Agent Collaboration</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Instantiate specialized autonomous agents to research, debate, and construct arguments directly on your dialogue canvas.
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6 hover:bg-accent hover:text-accent-foreground transition-colors">
            <h2 className="text-xl font-bold text-card-foreground mb-3">Advanced Layout Engine</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Organize your workspace exactly how you need it. Drag, drop, and snap tools into custom configurations. Your precise layout is securely stored to your user profile.
            </p>
          </div>

          <div className="bg-card border border-border rounded-lg p-6 hover:bg-accent hover:text-accent-foreground transition-colors">
            <h2 className="text-xl font-bold text-card-foreground mb-3">Issue Based Information System</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Map out wicked problems using the proven IBIS methodology. Break down complex decisions into Questions, Ideas, and Arguments.
            </p>
          </div>
        </div>

        <div className="mt-8 pt-6 border-t border-border text-center">
          <p className="text-sm text-muted-foreground">
            Open the Documentation tab or explore the Help menu to learn more about the advanced features.
          </p>
        </div>
      </div>
    </div>
  );
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
    <div className="h-screen w-screen flex items-center justify-center bg-gradient-to-br from-[#2D1B4E] via-[#4A2D73] to-[#1E293B] text-white font-sans overflow-hidden relative">
      {/* Decorative backdrop elements matching the warm sunset/violet theme */}
      <div className="absolute top-[-20%] left-[-10%] w-[70%] h-[70%] rounded-full bg-orange-500/20 blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-20%] right-[-10%] w-[70%] h-[70%] rounded-full bg-indigo-500/30 blur-[120px] pointer-events-none" />

      {/* Login Card - Glassmorphism */}
      <div className="w-[420px] p-8 bg-white/5 border border-white/10 rounded-[2rem] shadow-[0_8px_32px_0_rgba(0,0,0,0.3)] backdrop-blur-2xl relative z-10 flex flex-col gap-6 transition-all duration-300">
        <div className="text-center mt-2">
          <h1 className="text-5xl font-extrabold tracking-widest text-transparent bg-clip-text bg-gradient-to-r from-white via-orange-100 to-indigo-100 drop-shadow-md">NEXUS</h1>
          <p className="text-sm text-white/70 mt-3 font-light">Welcome back! Please sign in to continue.</p>
        </div>

        {error && (
          <div className="bg-red-500/20 border border-red-500/30 rounded-xl p-3 text-sm text-red-100 font-medium text-center shadow-inner">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-5 mt-2">
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium tracking-wide text-white/90">Email Address</label>
            <input 
              type="text" 
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="user@example.com"
              disabled={loading}
              className="px-4 py-3 bg-white/5 border border-white/10 rounded-xl text-sm text-white focus:outline-none focus:ring-2 focus:ring-orange-400/50 focus:border-orange-400/50 transition-all placeholder:text-white/30 shadow-inner"
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium tracking-wide text-white/90">Password</label>
            <input 
              type="password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              disabled={loading}
              className="px-4 py-3 bg-white/5 border border-white/10 rounded-xl text-sm text-white focus:outline-none focus:ring-2 focus:ring-orange-400/50 focus:border-orange-400/50 transition-all placeholder:text-white/30 shadow-inner"
            />
            <div className="text-right">
              <a href="#" className="text-[12px] text-indigo-300 hover:text-white transition-colors cursor-pointer">Forgot Password?</a>
            </div>
          </div>

          <button 
            type="submit"
            disabled={loading}
            className="w-full mt-4 py-3.5 bg-gradient-to-r from-orange-500 to-purple-500 text-white font-bold tracking-wide rounded-full hover:shadow-[0_0_20px_rgba(249,115,22,0.4)] disabled:opacity-50 disabled:cursor-not-allowed transition-all active:scale-[0.98] shadow-lg text-[15px]"
          >
            {loading ? 'SIGNING IN...' : 'SIGN IN'}
          </button>
        </form>

        <div className="pt-2 text-center mb-2">
          <p className="text-[13px] text-white/70">
            Don't have an account? <span className="text-indigo-300 hover:text-white cursor-pointer transition-colors font-medium">Sign Up</span>
          </p>
          <div className="mt-6 flex items-center gap-3">
            <div className="h-[1px] flex-1 bg-white/10" />
            <span className="text-[11px] text-white/50 tracking-wide uppercase">Or continue with</span>
            <div className="h-[1px] flex-1 bg-white/10" />
          </div>
          <div className="mt-5 flex justify-center gap-4">
             <div className="w-10 h-10 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center cursor-pointer transition-colors border border-white/10 shadow-sm">
               <span className="text-[15px] font-extrabold text-transparent bg-clip-text bg-gradient-to-br from-blue-400 via-red-400 to-yellow-400">G</span>
             </div>
             <div className="w-10 h-10 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center cursor-pointer transition-colors border border-white/10 shadow-sm">
               <span className="text-lg text-white"></span>
             </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [dbLayout, setDbLayout] = useState<any>(undefined);
  const saveTimeoutRef = useRef<any>(null);

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const base = import.meta.env.BASE_URL;
        const res = await fetch(`${base}api/auth/check`);
        const data: AuthCheckResponse = await res.json();
        if (data.authenticated) {
          setIsAuthenticated(true);
        }
      } catch (err) {
        console.error('Failed to verify session:', err);
      } finally {
        setLoading(false);
      }
    };
    checkAuth();
  }, []);

  const handleLoginSuccess = (_username: string) => {
    setIsAuthenticated(true);
  };

  useEffect(() => {
    if (isAuthenticated) {
      const base = import.meta.env.BASE_URL;
      fetch(`${base}api/layout`)
        .then(res => res.json())
        .then(data => {
          setDbLayout(data);

          // Fetch profile asynchronously on boot
          fetch(`${base}api/profile`)
            .then(r => r.json())
            .then(profile => {
              useUserProfileStore.getState().updateProfile({
                name: profile.full_name || 'Anonymous',
                role: profile.title || ''
              });
              if (profile.avatar_data) {
                useUserProfileStore.getState().setCustomAvatar(profile.avatar_data);
              }
            })
            .catch(console.error);

          // Initialize core registries
          initializeShell();

      // Register Custom Tab Components
      componentRegistry.register('nexus-dashboard', DashboardTab);
      componentRegistry.register('nexus-docs', DocsTab);
      componentRegistry.register('welcome', WelcomeTab);
      componentRegistry.register('user-profile', UserProfileTab);

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

      commandRegistry.registerCommand({
        id: 'app.open_welcome',
        label: 'Open Welcome',
        execute: () => {
          const layoutState = useLayoutStore.getState();
          layoutState.addTab('welcome', 'Welcome');
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

      menuRegistry.registerMenu('Help', {
        id: 'help.open_welcome',
        label: 'Welcome to Nexus',
        commandId: 'app.open_welcome',
      });

      // Automatically open the dashboard & docs tab on startup
      const layoutState = useLayoutStore.getState();
      layoutState.addTab('nexus-dashboard', 'Dashboard');
      layoutState.addTab('nexus-docs', 'Documentation');
        })
        .catch(err => {
          console.error("Failed to load layout from DB", err);
          setDbLayout({});
        });
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

  if (dbLayout === undefined) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-[#0D1117] text-emerald-500 font-mono text-sm">
        LOADING WORKSPACE LAYOUT FROM DATABASE...
      </div>
    );
  }

	return (
		<div style={{ width: '100vw', height: '100vh' }}>
			<NexusWorkspaceShell 
        disableLocalStorage={true}
        initialLayoutJson={dbLayout}
        onLayoutChange={(newLayout) => {
          if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
          saveTimeoutRef.current = setTimeout(() => {
            const base = import.meta.env.BASE_URL;
            fetch(`${base}api/layout`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ layout_data: newLayout })
            }).catch(err => console.error("Failed to save layout to DB", err));
          }, 1000);
        }}
      />
		</div>
	);
}

export default App;
